package message

import (
	"encoding/json"
	"fmt"

	"github.com/SherClockHolmes/webpush-go"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/config"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/cache"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/database"
	repo "github.com/TestingSDK2/sidekiq-backend/sidekiq-content/model"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/mongodatabase"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-models/model"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-models/model/notification"
	"github.com/pkg/errors"
	"github.com/sideshow/apns2"
	"github.com/sideshow/apns2/certificate"
	"github.com/sirupsen/logrus"
)

// Service - defines Chat service
type Service interface {
	FetchMessagesByGroup(boardID string, userID int) ([]*model.Chat, error)
	AddMessage(cache *cache.Cache, group model.Chat, profileID int) (*model.Chat, error)
	UpdateMessage(cache *cache.Cache, group model.Chat, profileID int) (map[string]interface{}, error)
	DeleteMessage(Id string, profileID int) (map[string]interface{}, error)
	FetchGroupMembers(id string, skipCache bool) ([]string, error)
	PushToMembers(groupMembers []string, msg *model.Chat, profileID int) error
}

type service struct {
	config    *config.Config
	dbMaster  *database.Database
	dbReplica *database.Database
	mongodb   *mongodatabase.DBConfig
	cache     *cache.Cache
}

// NewService - creates new Chat service
func NewService(repos *repo.Repos, conf *config.Config) Service {
	return &service{
		config:    conf,
		dbMaster:  repos.MasterDB,
		dbReplica: repos.ReplicaDB,
		mongodb:   repos.MongoDB,
		cache:     repos.Cache,
	}
}

// cache
func (s *service) FetchCachedGroupMembers(id string) *model.ChatGroup {
	key := getCacheKey(id)
	val, err := s.cache.GetValue(key)
	if err != nil {
		return nil
	}
	var group *model.ChatGroup
	json.Unmarshal([]byte(val), &group)
	return group
}

func (s *service) AddGroupToCache(discussion *model.ChatGroup) error {
	key := getCacheKey(discussion.Id.Hex())
	err := s.cache.SetValue(key, discussion.ToJSON())
	if err != nil {
		return err
	}
	s.cache.ExpireKey(key, cache.Expire18HR)
	return nil
}

func (s *service) FetchMessagesByGroup(boardID string, userID int) ([]*model.Chat, error) {
	return getMessagesByGroup(s.mongodb, boardID, userID)
}

func (s *service) AddMessage(cache *cache.Cache, group model.Chat, profileID int) (*model.Chat, error) {
	return addMessage(s.cache, s.mongodb, group, profileID)
}

func (s *service) UpdateMessage(cache *cache.Cache, group model.Chat, profileID int) (map[string]interface{}, error) {
	return updateMessage(s.cache, s.mongodb, group, profileID)
}

func (s *service) DeleteMessage(Id string, profileID int) (map[string]interface{}, error) {
	return deleteMessage(s.cache, s.mongodb, Id, profileID)
}

func (s *service) FetchGroupMembers(id string, skipCache bool) ([]string, error) {
	arr := make([]string, 0)
	if !skipCache {
		cachedGroupMembers := s.FetchCachedGroupMembers(id)
		if cachedGroupMembers != nil {
			// arr = append(arr, cachedGroupMembers.Admin)
			arr = append(arr, cachedGroupMembers.Members...)
			return arr, nil
		}
	}
	groupMembers, err := getGroupFromDB(s.mongodb, id)
	if err != nil {
		return nil, err
	}
	s.AddGroupToCache(groupMembers)
	// arr = append(arr, groupMembers.Admin)
	arr = append(arr, groupMembers.Members...)
	return arr, nil
}

func (s *service) PushToMembers(groupMembers []string, msg *model.Chat, profileID int) error {
	fmt.Println("reached PushToMembers.......")
	subs, err := getPushSubscriptionsByGroupMembers(s.dbMaster, groupMembers)
	if err != nil {
		return err
	}

	pushErrors := map[int]error{}
	fmt.Println(subs)
	for _, sub := range subs {
		fmt.Println(profileID, sub.ProfileID)
		if sub.ProfileID != profileID {
			subscription := sub.ToWebPush()
			notificationData := &notification.Message{
				Type:    "group",
				Content: msg.ToJSON(),
			}
			resp, err := webpush.SendNotification([]byte(notificationData.ToJSON()), subscription, &webpush.Options{
				Subscriber:      fmt.Sprintf("%d", sub.ProfileID),
				VAPIDPublicKey:  s.config.VapidPublicKey,
				VAPIDPrivateKey: s.config.VapidPrivateKey,
				TTL:             30,
			})
			if err != nil {
				pushErrors[sub.ID] = err
			}
			resp.Body.Close()
			// defer resp.Body.Close()
		}
	}

	fmt.Println("reached here......")

	if len(pushErrors) > 0 {
		return errors.New(fmt.Sprintf("Failed to send %d notifications", len(pushErrors)))
	}

	appleSubs, err := getApplePushSubscriptionsByGroupMembers(s.dbMaster, groupMembers)
	if err != nil {
		return err
	}

	if appleSubs != nil && len(appleSubs) > 0 {
		cert, err := certificate.FromP12File("./apns/Apple_website_aps_production.p12", "")
		if err != nil {
			return errors.Wrap(err, "Failed to read Apple_website_aps_production.p12 cert from file")
		}
		client := apns2.NewClient(cert).Production()

		fmt.Println("applesubs len: ", len(appleSubs))
		for _, sub := range appleSubs {
			fmt.Println("sub: ", sub)
			// payload := &notification.Message{
			// 	Type:    "discussion",
			// 	Content: msg.ToJSON(),
			// }
			if sub.UserID != profileID {
				data := &map[string]interface{}{
					"aps": &map[string]interface{}{
						"alert": &map[string]interface{}{
							"title":  "Sidkiq - Discussion",
							"body":   fmt.Sprintf("Discussion: %d", msg.Id),
							"action": "View",
						},
						"url-args": []string{"team/discussion", msg.GroupID.Hex()},
					},
				}
				payload, err := json.Marshal(data)
				fmt.Println("payload: ", string(payload))

				apnsNotification := &apns2.Notification{}
				apnsNotification.DeviceToken = sub.DeviceToken
				apnsNotification.Topic = "web.com.sidekiq.app"
				apnsNotification.Payload = []byte(payload)

				res, err := client.Push(apnsNotification)
				if err != nil {
					fmt.Println("error", err) // error Post "https://api.push.apple.com/3/device/F2EA0D7B3509E4468B1FF5B03BDEE29366A25D1EA13ECCEB44249043DBF7E7DC": remote error: tls: expired certificate
					pushErrors[sub.UserID] = err
				} else {
					logrus.Printf("%v %v %v\n", res.StatusCode, res.ApnsID, res.Reason)
				}
			}
		}

		if len(pushErrors) > 0 {
			return errors.New(fmt.Sprintf("Failed to send %d notifications", len(pushErrors)))
		}
	}

	return nil
}
