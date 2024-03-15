package util

import "fmt"

var PREFIX string = "sidekiq"

/* ************* USER ************* */
func GetKeyForUserImage(accountID int, size string) string {
	if size == "" {
		return fmt.Sprintf("%s/user/%d/media/", PREFIX, accountID)
	} else if size == "thumbs" {
		return fmt.Sprintf("%s/user/%d/media/thumbs/", PREFIX, accountID)
	}

	return fmt.Sprintf("%s/user/%d/media/thumbs/%s/", PREFIX, accountID, size)
}

func GetKeyForUserQR(userID int) string {
	return fmt.Sprintf("%s/user/%d/temp/", PREFIX, userID)
}

/* ************* PROFILE ************* */
func GetKeyForProfileImage(accountID, profileID int, size string) string {
	if size == "" {
		return fmt.Sprintf("%s/user/%d/profile/%d/media/", PREFIX, accountID, profileID)
	} else if size == "thumbs" {
		return fmt.Sprintf("%s/user/%d/profile/%d/media/thumbs/", PREFIX, accountID, profileID)
	}
	return fmt.Sprintf("%s/user/%d/profile/%d/media/thumbs/%s/", PREFIX, accountID, profileID, size)
}

func GetKeyForProfileQR(userID, profileID int) string {
	return fmt.Sprintf("%s/user/%d/profile/%d/temp/", PREFIX, userID, profileID)
}

/* ************* BOARD ************* */
func GetKeyForBoardCover(userID, profileID int, boardID, size string) string {
	if size == "" {
		return fmt.Sprintf("%s/user/%d/profile/%d/board/%s/cover/", PREFIX, userID, profileID, boardID)
	} else if size == "thumbs" {
		return fmt.Sprintf("%s/user/%d/profile/%d/board/%s/cover/thumbs/", PREFIX, userID, profileID, boardID)
	}
	return fmt.Sprintf("%s/user/%d/profile/%d/board/%s/cover/thumbs/%s/", PREFIX, userID, profileID, boardID, size)
}

func GetKeyForBoardMedia(userID, profileID int, boardID, size string) string {
	if size == "" {
		return fmt.Sprintf("/user/%d/profile/%d/board/%s/media/", userID, profileID, boardID)
	} else if size == "thumbs" {
		return fmt.Sprintf("/user/%d/profile/%d/board/%s/media/thumbs/", userID, profileID, boardID)
	}
	return fmt.Sprintf("/user/%d/profile/%d/board/%s/media/thumbs/%s/", userID, profileID, boardID, size)
}

func GetKeyForBoardPostMedia(userID, profileID int, boardID, postID, size string) string {
	if size == "" {
		return fmt.Sprintf("/user/%d/profile/%d/board/%s/post/%s/media/", userID, profileID, boardID, postID)
	} else if size == "thumbs" {
		return fmt.Sprintf("/user/%d/profile/%d/board/%s/post/%s/media/thumbs/", userID, profileID, boardID, postID)
	}
	return fmt.Sprintf("/user/%d/profile/%d/board/%s/post/%s/media/thumbs/%s/", userID, profileID, boardID, postID, size)
}

func GetKeyForBoardMediaForFetch(userID, profileID int, boardID, size string) string {
	if size == "" {
		return fmt.Sprintf("user/%d/profile/%d/board/%s/media/", userID, profileID, boardID)
	} else if size == "thumbs" {
		return fmt.Sprintf("user/%d/profile/%d/board/%s/media/thumbs/", userID, profileID, boardID)
	}
	return fmt.Sprintf("user/%d/profile/%d/board/%s/media/thumbs/%s/", userID, profileID, boardID, size)
}

func GetKeyForCollectionMedia(boardID, collectionID string) string {
	return fmt.Sprintf("/board/%s/collection/%s/", boardID, collectionID)
}

func GetKeyForPostCollectionMedia(userID, profileID int, boardID, postID, collectionID string, size string) string {
	if size == "" {
		return fmt.Sprintf("/user/%d/profile/%d/board/%s/post/%s/collection/%s/", userID, profileID, boardID, postID, collectionID)
	} else if size == "thumbs" {
		return fmt.Sprintf("/user/%d/profile/%d/board/%s/post/%s/collection/%s/thumbs/", userID, profileID, boardID, postID, collectionID)
	}
	return fmt.Sprintf("/user/%d/profile/%d/board/%s/post/%s/collection/%s/thumbs/%s/", userID, profileID, boardID, postID, collectionID, size)
}

/* ************* ORGANIZATION ************* */
func GetKeyForOrganizationImage(accountID int, size string) string {
	if size == "" {
		return fmt.Sprintf("%s/%d/media/org/", PREFIX, accountID)
	} else if size == "thumbs" {
		return fmt.Sprintf("%s/%d/media/org/thumbs/", PREFIX, accountID)
	}
	return fmt.Sprintf("%s/%d/media/org/thumbs/%s/", PREFIX, accountID, size)
}

func GetKeyForPostCover(accountID, profileID int, boardID, postID, size string) string {
	if size == "" {
		return fmt.Sprintf("user/%d/profile/%d/board/%s/post/%s/cover/", accountID, profileID, boardID, postID)
	} else if size == "thumbs" {
		return fmt.Sprintf("user/%d/profile/%d/board/%s/post/%s/cover/thumbs/", accountID, profileID, boardID, postID)
	}
	return fmt.Sprintf("user/%d/profile/%d/board/%s/post/%s/cover/thumbs/%s/", accountID, profileID, boardID, postID, size)
}

func MovePostKey(accountID, profileID int, boardID string) string {
	return fmt.Sprintf("user/%d/profile/%d/board/%s/post/", accountID, profileID, boardID)
}
