package apns

import (
	"archive/zip"
	"crypto"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/ProImaging/sidekiq-backend/sidekiq-content/cache"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/database"
	"github.com/ProImaging/sidekiq-backend/sidekiq-models/model"
	"github.com/ProImaging/sidekiq-backend/sidekiq-models/model/notification"
	"github.com/pkg/errors"
	"go.mozilla.org/pkcs7"
	"golang.org/x/crypto/pkcs12"
)

func (s *service) FetchPushSubscriptions(profileID int) []*notification.ApplePushSubscription {
	return fetchPushSubscriptions(s.cache, s.dbMaster, profileID)
}

func (s *service) CreatePushSubscription(profileID int, deviceToken string) (int, error) {
	sub := &notification.ApplePushSubscription{
		UserID:      profileID,
		DeviceToken: deviceToken,
	}
	return insertPushSubscriptions(s.dbMaster, sub)
}

func (s *service) RemovePushSubscription(profileID int, deviceToken string) error {
	sub := &notification.ApplePushSubscription{
		UserID:      profileID,
		DeviceToken: deviceToken,
	}
	return removePushSubscription(s.dbMaster, sub)
}

func (s *service) GeneratePushPackage(user *model.Account) (string, error) {
	locPushPack, err := s.tmpFileStore.MakeFolder("", user.ID, "Sidekiq.pushpackage")
	if err != nil {
		return "", errors.Wrap(err, "Error creating pushpackage folder")
	}

	// Prepare website.json
	websiteFile, err := ioutil.ReadFile("./apns/Sidekiq.pushpackage/website.json")
	if err != nil {
		return "", err
	}

	tmpl, err := template.New("website.json").Parse(string(websiteFile))
	if err != nil {
		return "", err
	}

	data := map[string]string{
		"AuthenticationToken": user.Token,
	}
	buf := new(strings.Builder)
	err = tmpl.Execute(buf, data)

	err = ioutil.WriteFile(path.Join(locPushPack, "website.json"), []byte(buf.String()), 0644)
	if err != nil {
		return "", errors.Wrap(err, "Error writing website.json")
	}

	// Prepare manifest.json
	manifestFile, err := ioutil.ReadFile("./apns/Sidekiq.pushpackage/manifest.json")
	if err != nil {
		return "", err
	}

	tmpl, err = template.New("manifest.json").Parse(string(manifestFile))
	if err != nil {
		return "", err
	}
	sum := sha512.Sum512([]byte(buf.String()))
	data = map[string]string{
		"WebsiteHash": hex.EncodeToString(sum[:64]),
	}
	buf = new(strings.Builder)
	err = tmpl.Execute(buf, data)

	err = ioutil.WriteFile(path.Join(locPushPack, "manifest.json"), []byte(buf.String()), 0644)
	if err != nil {
		return "", errors.Wrap(err, "Error writing manifest.json")
	}

	// Prepare signature
	signedData, err := pkcs7.NewSignedData([]byte(buf.String()))
	if err != nil {
		return "", errors.Wrap(err, "Error initializing pkcs7 signed data")
	}

	asn1data, err := ioutil.ReadFile("./apns/Apple_website_aps_production.p12")
	if err != nil {
		return "", errors.Wrap(err, "Error reading Apple_website_aps_production.p12")
	}

	key, cert, err := pkcs12.Decode(asn1data, "")
	if err != nil {
		return "", errors.Wrap(err, "Error Parsing Certificate")
	}
	err = signedData.AddSigner(cert, crypto.PrivateKey(key), pkcs7.SignerInfoConfig{})
	if err != nil {
		return "", errors.Wrap(err, "Error Adding Signer Certificate")
	}
	signedData.Detach()

	detachedSignature, err := signedData.Finish()
	if err != nil {
		return "", errors.Wrap(err, "Error finish signing data")
	}

	err = ioutil.WriteFile(path.Join(locPushPack, "signature"), detachedSignature, 0644)
	if err != nil {
		return "", errors.Wrap(err, "Error writting signature")
	}

	// Copy Icons Folder
	locIcons, err := s.tmpFileStore.MakeFolder("", user.ID, "Sidekiq.pushpackage/icon.iconset")
	if err != nil {
		return "", errors.Wrap(err, "Error creating icon.iconset folder")
	}
	err = CopyFolder("./apns/Sidekiq.pushpackage/icon.iconset", locIcons)
	if err != nil {
		return "", errors.Wrap(err, "Error copying the icon.iconset folder")
	}

	// Zip Folder
	return ZipFolder(locPushPack+"/", locPushPack+".zip")
}

func ZipFolder(srcDir string, dest string) (string, error) {
	outFile, err := os.Create(dest)
	if err != nil {
		return "", errors.Wrap(err, "Error creating zip file")
	}
	defer outFile.Close()

	w := zip.NewWriter(outFile)

	err = AddFilesToZip(w, srcDir, "")
	if err != nil {
		return "", errors.Wrap(err, "Error add files to zip")
	}

	err = w.Close()
	if err != nil {
		return "", errors.Wrap(err, "Error closing zip file")
	}
	return dest, nil
}

func AddFilesToZip(w *zip.Writer, basePath string, baseInZip string) error {
	entries, err := ioutil.ReadDir(basePath)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("Error reading directory: %s", basePath))
	}

	for _, entry := range entries {
		if entry.IsDir() {
			newBase := basePath + entry.Name() + "/"
			err = AddFilesToZip(w, newBase, baseInZip+entry.Name()+"/")
			if err != nil {
				return err
			}
		} else {

			dat, err := ioutil.ReadFile(basePath + entry.Name())
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("Error reading file: %s", entry.Name()))
			}

			// Add some files to the archive.
			f, err := w.Create(baseInZip + entry.Name())
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("Error adding file to archive: %s", entry.Name()))
			}
			_, err = f.Write(dat)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("Error writting file to archive: %s", entry.Name()))
			}
		}
	}
	return nil
}

func CopyFolder(srcDir string, dest string) error {
	entries, err := ioutil.ReadDir(srcDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		sourcePath := filepath.Join(srcDir, entry.Name())
		destPath := filepath.Join(dest, entry.Name())

		if err := CopyFile(sourcePath, destPath); err != nil {
			return err
		}
		// if err := os.Lchown(destPath, int(stat.Uid), int(stat.Gid)); err != nil {
		// 	return err
		// }
	}
	return nil
}

func CopyFile(srcFile, dstFile string) error {
	out, err := os.Create(dstFile)
	if err != nil {
		return err
	}

	defer out.Close()

	in, err := os.Open(srcFile)
	defer in.Close()
	if err != nil {
		return err
	}

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}

	return nil
}

func fetchPushSubscriptions(cache *cache.Cache, db *database.Database, profileID int) []*notification.ApplePushSubscription {
	stmt := "SELECT id, profileID, type, deviceToken, expirationTime, createDate FROM `sidekiq-dev`.PushSubscriptionsApple WHERE profileID = ?"
	subscriptions := []*notification.ApplePushSubscription{}
	db.Conn.Select(&subscriptions, stmt, profileID)
	return subscriptions
}

func insertPushSubscriptions(db *database.Database, subscription *notification.ApplePushSubscription) (int, error) {
	stmt := "INSERT INTO `sidekiq-dev`.PushSubscriptionsApple (profileID, type, deviceToken, expirationTime) VALUES(:profileID,:type,:deviceToken,:expirationTime);"
	r, err := db.Conn.NamedExec(stmt, subscription)
	if err != nil {
		return 0, err
	}
	id, err := r.LastInsertId()
	if err != nil {
		return 0, errors.Wrap(err, "Failed to get LastInsertId")
	}
	return int(id), nil
}

func removePushSubscription(db *database.Database, subscription *notification.ApplePushSubscription) error {
	stmt := "DELETE FROM `sidekiq-dev`.PushSubscriptionsApple WHERE profileID = :profileID AND deviceToken = :deviceToken;"
	_, err := db.Conn.NamedExec(stmt, subscription)
	return err
}
