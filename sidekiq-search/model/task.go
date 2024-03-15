package model

import (
	"encoding/json"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Task struct {
	Id                 primitive.ObjectID `json:"_id" bson:"_id"`
	PostID             primitive.ObjectID `json:"postID" bson:"postID"`
	Type               string             `json:"type" bson:"type"`
	Description        string             `json:"description" bson:"description"`
	SubTasks           []SubTask          `json:"subTasks" bson:"subTasks"`
	Title              string             `json:"title" bson:"title"`
	TaskPriority       string             `json:"taskPriority" bson:"taskPriority"`
	AssignedToID       string             `json:"assignedToID" bson:"assignedToID"`
	TaskStatus         string             `json:"taskStatus" bson:"taskStatus"`
	ReminderTimer      time.Time          `json:"reminderTimer" bson:"reminderTimer"`
	DueDate            string             `json:"dueDate" bson:"dueDate"`
	DueTime            string             `json:"dueTime" bson:"dueTime"`
	CompleteDate       time.Time          `json:"completeDate" bson:"completeDate"`
	ArchiveDate        time.Time          `json:"archiveDate" bson:"archiveDate"`
	AssignedMemberInfo *ConciseProfile    `json:"assignedMemberInfo"`
	Likes              []string           `json:"likes" db:"likes"`
	TotalComments      int                `json:"totalComments" bson:"totalComments"`
	TotalLikes         int                `json:"totalLikes" db:"totalLikes"`
	IsLiked            bool               `json:"isLiked" db:"isLiked"`
	Comments           []Comment          `json:"comments" bson:"comments"`
	EditBy             string             `json:"editBy" bson:"editBy"`
	EditDate           *time.Time         `json:"editDate" bson:"editDate"`
}

type SubTask struct {
	Task       string    `json:"task" bson:"task"`
	Status     string    `json:"status" bson:"status"`
	ChangeDate time.Time `json:"changeDate" bson:"changeDate"`
}

func (b Task) ToMap() (dat map[string]interface{}) {
	d, _ := json.Marshal(b)
	json.Unmarshal(d, &dat)
	return
}
