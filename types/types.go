package types

import (
	"config-service/utils/consts"
	"time"

	"github.com/armosec/armoapi-go/armotypes"
	"github.com/armosec/armoapi-go/configservice"
	"github.com/armosec/armoapi-go/notifications"
	"github.com/armosec/armosec-infra/kdr"
	opapolicy "github.com/kubescape/opa-utils/reporthandling"
	uuid "github.com/satori/go.uuid"
)

// Document - document in db
type Document[T DocContent] struct {
	ID        string   `json:"_id" bson:"_id"`
	Customers []string `json:"customers" bson:"customers"`
	Content   T        `json:",inline" bson:"inline"`
}

// NewDocument - create new document per doc content T
func NewDocument[T DocContent](content T, customerGUID string) Document[T] {
	content.InitNew()
	content.SetGUID(uuid.NewV4().String())
	content.SetUpdatedTime(nil)
	doc := Document[T]{
		ID:      content.GetGUID(),
		Content: content,
	}
	if customerGUID != "" {
		doc.Customers = append(doc.Customers, customerGUID)
	}
	return doc
}

// Doc Content interface for data types embedded in DB documents
type DocContent interface {
	*CustomerConfig | *Cluster | *PostureExceptionPolicy | *VulnerabilityExceptionPolicy | *Customer |
		*Framework | *Repository | *RegistryCronJob | *CollaborationConfig | *Cache | *ClusterAttackChainState | *AggregatedVulnerability |
		*RuntimeIncident | *RuntimeAlert | *IntegrationReference
	InitNew()
	GetReadOnlyFields() []string
	//default implementation exist in portal base
	GetName() string
	SetName(name string)
	GetGUID() string
	SetGUID(guid string)
	GetAttributes() map[string]interface{}
	SetAttributes(attributes map[string]interface{})
	SetUpdatedTime(updatedTime *time.Time)
	GetUpdatedTime() *time.Time
	GetCreationTime() *time.Time
}

// redefine types for Doc Content implementations

// DocContent implementations

type AggregatedVulnerability struct {
	notifications.AggregatedVulnerability `json:",inline" bson:",inline"`
	//needed only for tests
	Name string `json:"name,omitempty" bson:"name,omitempty"`
}

func (v *AggregatedVulnerability) GetGUID() string {
	return v.GUID
}

func (v *AggregatedVulnerability) SetGUID(guid string) {
	v.GUID = guid
}

func (v *AggregatedVulnerability) GetCreationTime() *time.Time {
	if v.CreationTime == "" {
		return nil
	}
	creationTime, err := time.Parse(time.RFC3339, v.CreationTime)
	if err != nil {
		return nil
	}
	return &creationTime
}

func (v *AggregatedVulnerability) SetUpdatedTime(updatedTime *time.Time) {
	if updatedTime == nil {
		v.UpdatedTime = time.Now().UTC().Format(time.RFC3339)
		return
	}
	v.UpdatedTime = updatedTime.UTC().Format(time.RFC3339)
}

func (v *AggregatedVulnerability) GetUpdatedTime() *time.Time {
	if v.UpdatedTime == "" {
		return nil
	}
	updatedTime, err := time.Parse(time.RFC3339, v.UpdatedTime)
	if err != nil {
		return nil
	}
	return &updatedTime
}
func (c *AggregatedVulnerability) GetReadOnlyFields() []string {
	return commonReadOnlyFieldsV1
}

func (v *AggregatedVulnerability) InitNew() {
	v.CreationTime = time.Now().UTC().Format(time.RFC3339)
}
func (v *AggregatedVulnerability) GetName() string {
	return v.Name
}

func (v *AggregatedVulnerability) SetName(name string) {
	v.Name = name
}

func (v *AggregatedVulnerability) GetAttributes() map[string]interface{} { return nil }

func (v *AggregatedVulnerability) SetAttributes(attributes map[string]interface{}) {}

type CollaborationConfig notifications.CollaborationConfig

func (p *CollaborationConfig) GetReadOnlyFields() []string {
	return commonReadOnlyFieldsAllowRename
}
func (p *CollaborationConfig) InitNew() {
	p.CreationTime = time.Now().UTC().Format(time.RFC3339)
}

func (p *CollaborationConfig) GetCreationTime() *time.Time {
	if p.CreationTime == "" {
		return nil
	}
	creationTime, err := time.Parse(time.RFC3339, p.CreationTime)
	if err != nil {
		return nil
	}
	return &creationTime
}

type CustomerConfig struct {
	armotypes.CustomerConfig `json:",inline" bson:"inline"`
	GUID                     string `json:"guid" bson:"guid"`
	CreationTime             string `json:"creationTime" bson:"creationTime"`
	UpdatedTime              string `json:"updatedTime" bson:"updatedTime"`
}

func (c *CustomerConfig) GetGUID() string {
	return c.GUID
}

func (c *CustomerConfig) SetGUID(guid string) {
	c.GUID = guid
}
func (c *CustomerConfig) GetName() string {
	if c.Name == "" &&
		c.Scope.Attributes != nil &&
		c.Scope.Attributes["cluster"] != "" {
		return c.Scope.Attributes["cluster"]
	}
	return c.Name
}
func (c *CustomerConfig) SetName(name string) {
	c.Name = name
}
func (c *CustomerConfig) GetReadOnlyFields() []string {
	return commonReadOnlyFieldsV1
}
func (c *CustomerConfig) InitNew() {
	c.CreationTime = time.Now().UTC().Format(time.RFC3339)
	if c.Scope.Attributes != nil && c.Scope.Attributes["cluster"] != "" {
		c.Name = c.Scope.Attributes["cluster"]
	}
}
func (c *CustomerConfig) GetAttributes() map[string]interface{} {
	return c.Attributes
}
func (c *CustomerConfig) SetAttributes(attributes map[string]interface{}) {
	c.Attributes = attributes
}

func (c *CustomerConfig) SetUpdatedTime(updatedTime *time.Time) {
	if updatedTime == nil {
		c.UpdatedTime = time.Now().UTC().Format(time.RFC3339)
		return
	}
	c.UpdatedTime = updatedTime.UTC().Format(time.RFC3339)
}

func (p *CustomerConfig) GetUpdatedTime() *time.Time {
	if p.UpdatedTime == "" {
		return nil
	}
	updatedTime, err := time.Parse(time.RFC3339, p.UpdatedTime)
	if err != nil {
		return nil
	}
	return &updatedTime
}

func (p *CustomerConfig) GetCreationTime() *time.Time {
	if p.CreationTime == "" {
		return nil
	}
	creationTime, err := time.Parse(time.RFC3339, p.CreationTime)
	if err != nil {
		return nil
	}
	return &creationTime
}

// DocContent implementations

type Framework opapolicy.Framework

func (*Framework) GetReadOnlyFields() []string {
	return commonReadOnlyFields
}
func (f *Framework) InitNew() {
	f.CreationTime = time.Now().UTC().Format(time.RFC3339)
}

func (f *Framework) GetCreationTime() *time.Time {
	if f.CreationTime == "" {
		return nil
	}
	creationTime, err := time.Parse(time.RFC3339, f.CreationTime)
	if err != nil {
		return nil
	}
	return &creationTime
}

type Customer configservice.PortalCustomer

func (c *Customer) GetReadOnlyFields() []string {
	return commonReadOnlyFields
}
func (c *Customer) InitNew() {
	c.SubscriptionDate = time.Now().UTC().Format(time.RFC3339)
}
func (c *Customer) GetCreationTime() *time.Time {
	if c.SubscriptionDate == "" {
		return nil
	}
	creationTime, err := time.Parse(time.RFC3339, c.SubscriptionDate)
	if err != nil {
		return nil
	}
	return &creationTime
}

type Cluster armotypes.PortalCluster

func (c *Cluster) GetReadOnlyFields() []string {
	return clusterReadOnlyFields
}
func (c *Cluster) InitNew() {
	c.SubscriptionDate = time.Now().UTC().Format(time.RFC3339)
	if c.Attributes == nil {
		c.Attributes = make(map[string]interface{})
	}
}

func (c *Cluster) GetCreationTime() *time.Time {
	if c.SubscriptionDate == "" {
		return nil
	}
	creationTime, err := time.Parse(time.RFC3339, c.SubscriptionDate)
	if err != nil {
		return nil
	}
	return &creationTime
}

type VulnerabilityExceptionPolicy armotypes.VulnerabilityExceptionPolicy

func (c *VulnerabilityExceptionPolicy) GetReadOnlyFields() []string {
	return commonReadOnlyFieldsV1
}
func (c *VulnerabilityExceptionPolicy) InitNew() {
	c.CreationTime = time.Now().UTC().Format(time.RFC3339)
}
func (c *VulnerabilityExceptionPolicy) GetCreationTime() *time.Time {
	if c.CreationTime == "" {
		return nil
	}
	creationTime, err := time.Parse(time.RFC3339, c.CreationTime)
	if err != nil {
		return nil
	}
	return &creationTime
}

type PostureExceptionPolicy armotypes.PostureExceptionPolicy

func (p *PostureExceptionPolicy) GetReadOnlyFields() []string {
	return commonReadOnlyFieldsV1
}
func (p *PostureExceptionPolicy) InitNew() {
	p.CreationTime = time.Now().UTC().Format(time.RFC3339)
}

func (p *PostureExceptionPolicy) GetCreationTime() *time.Time {
	if p.CreationTime == "" {
		return nil
	}
	creationTime, err := time.Parse(time.RFC3339, p.CreationTime)
	if err != nil {
		return nil
	}
	return &creationTime
}

type Repository armotypes.PortalRepository

func (*Repository) GetReadOnlyFields() []string {
	return repositoryReadOnlyFields
}
func (r *Repository) InitNew() {
	r.CreationDate = time.Now().UTC().Format(time.RFC3339)
	if r.Attributes == nil {
		r.Attributes = make(map[string]interface{})
	}
}

func (r *Repository) GetCreationTime() *time.Time {
	if r.CreationDate == "" {
		return nil
	}
	creationTime, err := time.Parse(time.RFC3339, r.CreationDate)
	if err != nil {
		return nil
	}
	return &creationTime
}

type RegistryCronJob armotypes.PortalRegistryCronJob

func (*RegistryCronJob) GetReadOnlyFields() []string {
	return croneJobReadOnlyFields
}

func (r *RegistryCronJob) InitNew() {
	r.CreationDate = time.Now().UTC().Format(time.RFC3339)
	if r.Attributes == nil {
		r.Attributes = make(map[string]interface{})
	}
}

func (r *RegistryCronJob) GetCreationTime() *time.Time {
	if r.CreationDate == "" {
		return nil
	}
	creationTime, err := time.Parse(time.RFC3339, r.CreationDate)
	if err != nil {
		return nil
	}
	return &creationTime
}

type ClusterAttackChainState armotypes.ClusterAttackChainState

func (c *ClusterAttackChainState) GetReadOnlyFields() []string {
	return attackChainReadOnlyFields
}
func (c *ClusterAttackChainState) InitNew() {
	c.CreationTime = time.Now().UTC().Format(time.RFC3339)
}

func (c *ClusterAttackChainState) SetGUID(guid string) {
	c.GUID = guid
}

func (c *ClusterAttackChainState) GetGUID() string {
	return c.GUID
}

func (c *ClusterAttackChainState) GetCreationTime() *time.Time {
	if c.CreationTime == "" {
		return nil
	}
	creationTime, err := time.Parse(time.RFC3339, c.CreationTime)
	if err != nil {
		return nil
	}
	return &creationTime
}

type VulnerabilityExceptionsSeverityUpdate struct {
	Cves          []string `json:"cves" binding:"required"`
	SeverityScore int      `json:"severityScore" binding:"required"`
}

type PostureExceptionsSeverityUpdate struct {
	ControlIDS    []string `json:"controlIDS" binding:"required"`
	SeverityScore int      `json:"severityScore" binding:"required"`
}

type RuntimeIncident struct {
	kdr.RuntimeIncident `json:",inline" bson:",inline"`
	CreationDayDate     *time.Time `json:"creationDayDate,omitempty" bson:"creationDayDate,omitempty"`
	ResolveDayDate      *time.Time `json:"resolveDayDate,omitempty" bson:"resolveDayDate,omitempty"`
}

var runtimeIncidentReadOnlyFields = append([]string{"creationTimestamp", "creationDayDate"}, commonReadOnlyFieldsV1...)

func (r *RuntimeIncident) GetReadOnlyFields() []string {
	readOnlyFields := runtimeIncidentReadOnlyFields
	if r.RelatedAlerts == nil || len(r.RelatedAlerts) == 0 {
		readOnlyFields = append(readOnlyFields, "relatedAlerts")
	}
	return readOnlyFields
}

func (r *RuntimeIncident) InitNew() {
	r.CreationTimestamp = time.Now().UTC()
	cDayDate := time.Date(r.CreationTimestamp.Year(), r.CreationTimestamp.Month(), r.CreationTimestamp.Day(), 0, 0, 0, 0, time.UTC)
	r.CreationDayDate = &cDayDate
}

func (r *RuntimeIncident) GetCreationTime() *time.Time {
	return &r.CreationTimestamp
}

func (r *RuntimeIncident) SetGUID(guid string) {
	//if user defined a GUID do not change it
	//alow only to reset it to empty
	if r.GUID == "" || guid == "" {
		r.GUID = guid
	}
}

type RuntimeAlert struct {
	armotypes.PortalBase `json:",inline" bson:"inline"`
	kdr.RuntimeAlert     `json:",inline" bson:"inline"`
}

func (r *RuntimeAlert) GetReadOnlyFields() []string {
	return runtimeIncidentReadOnlyFields
}

func (r *RuntimeAlert) InitNew() {
	r.Timestamp = time.Now().UTC()
}

func (r *RuntimeAlert) GetCreationTime() *time.Time {
	return &r.Timestamp
}

func (r *RuntimeAlert) SetGUID(guid string) {

}

type IntegrationReference notifications.IntegrationReference

func (i *IntegrationReference) GetReadOnlyFields() []string {
	return commonReadOnlyFieldsV1
}

func (i *IntegrationReference) InitNew() {
	i.CreationTime = time.Now().UTC()
}

func (i *IntegrationReference) GetCreationTime() *time.Time {
	return &i.CreationTime
}

var baseReadOnlyFields = []string{consts.IdField, consts.GUIDField}
var commonReadOnlyFields = append([]string{consts.NameField}, baseReadOnlyFields...)
var commonReadOnlyFieldsV1 = append([]string{"creationTime"}, commonReadOnlyFields...)
var commonReadOnlyFieldsAllowRename = append([]string{"creationTime"}, baseReadOnlyFields...)
var clusterReadOnlyFields = append([]string{"subscription_date"}, commonReadOnlyFields...)
var repositoryReadOnlyFields = append([]string{"creationDate"}, commonReadOnlyFields...)
var croneJobReadOnlyFields = append([]string{"creationTime", "clusterName", "registryName"}, commonReadOnlyFields...)
var attackChainReadOnlyFields = append([]string{"creationTime", "customerGUID", "clusterName"}, commonReadOnlyFieldsV1...)
