package types

import (
	"encoding/json"
	"time"

	"github.com/armosec/armoapi-go/armotypes"
)

type Cache armotypes.PortalCache[json.RawMessage]

func (c *Cache) GetReadOnlyFields() []string {
	return commonReadOnlyFieldsV1
}
func (c *Cache) InitNew() {
	c.CreationTime = time.Now().UTC().Format(time.RFC3339)
}
func (c *Cache) GetAttributes() map[string]interface{} {
	return nil
}
func (c *Cache) SetAttributes(attributes map[string]interface{}) {
}
func (c *Cache) GetGUID() string {
	return c.GUID
}
func (c *Cache) SetGUID(guid string) {
	//if user defined a GUID do not change it
	//alow only to reset it to empty
	if c.GUID == "" || guid == "" {
		c.GUID = guid
	}
}
func (c *Cache) GetName() string {
	return c.Name
}
func (c *Cache) SetName(name string) {
	c.Name = name
}
func (c *Cache) GetCreationTime() *time.Time {
	if c.CreationTime == "" {
		return nil
	}
	creationTime, err := time.Parse(time.RFC3339, c.CreationTime)
	if err != nil {
		return nil
	}
	return &creationTime
}
func (c *Cache) GetUpdatedTime() *time.Time {
	if c.UpdatedTime == "" {
		return nil
	}
	UpdatedTime, err := time.Parse(time.RFC3339, c.UpdatedTime)
	if err != nil {
		return nil
	}
	return &UpdatedTime
}
func (c *Cache) SetUpdatedTime(updatedTime *time.Time) {
	if updatedTime == nil {
		c.UpdatedTime = time.Now().UTC().Format(time.RFC3339)
		return
	}
	c.UpdatedTime = updatedTime.UTC().Format(time.RFC3339)
}
func (c *Cache) SetExpiryTime(expiryTime time.Time) {
	(*armotypes.PortalCache[json.RawMessage])(c).SetExpiryTime(expiryTime)
}

func (c *Cache) SetTTL(ttl time.Duration) {
	(*armotypes.PortalCache[json.RawMessage])(c).SetTTL(ttl)
}

func (c *Cache) GetTimestampFieldName() string {
	return "creationTime"
}
