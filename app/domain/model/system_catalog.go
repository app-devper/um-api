package model

import "time"

// SystemCatalogEntry describes a System once, whichever Clients have it: its
// name, the front end users open (WebUrl) and the API a new Client starts
// with (Host).
type SystemCatalogEntry struct {
	SystemCode  string    `bson:"_id" json:"systemCode"`
	SystemName  string    `bson:"systemName" json:"systemName"`
	WebUrl      string    `bson:"webUrl" json:"webUrl"`
	Host        string    `bson:"host" json:"host"`
	UpdatedDate time.Time `bson:"updatedDate" json:"-"`
}
