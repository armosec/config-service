package consts

const (

	//Context keys for stored values
	DocContentKey  = "docContent"           //key for doc content from request body
	CustomerGUID   = "customerGUID"         //key for customer GUID from request login details
	Collection     = "collection"           //key for db collection name of the request
	ReqLogger      = "reqLogger"            //key for request logger
	AdminAccess    = "adminAccess"          //key for admin access flag
	BodyDecoder    = "customBodyDecoder"    //key for custom body decoder
	ResponseSender = "customResponseSender" //key for custom response sender
	PutDocFields   = "customPutDocFields"   //key for string list of fields name to update in PUT requests, only these fields will be updated
	SchemaInfo     = "schemaInfo"           //key for schema info
	BaseDocID      = "baseDocID"            //key for base document ID, for pagination over nested documents

	//PATHS
	ClusterPath                           = "/cluster"
	PostureExceptionPolicyPath            = "/v1_posture_exception_policy"
	VulnerabilityExceptionPolicyPath      = "/v1_vulnerability_exception_policy"
	CustomerConfigPath                    = "/v1_customer_configuration"
	FrameworkPath                         = "/v1_opa_framework"
	RepositoryPath                        = "/v1_repository"
	AdminPath                             = "/v1_admin"
	CustomerPath                          = "/customer"
	TenantPath                            = "/customer_tenant"
	RegistryCronJobPath                   = "/v1_registry_cron_job"
	NotificationConfigPath                = "/v1_notification_config"
	CustomerStatePath                     = "/v1_customer_state"
	ActiveSubscriptionPath                = "/v1_active_subscription"
	CollaborationConfigPath               = "/v1_collaboration_configuration"
	UsersNotificationsCachePath           = "/v1_users_notifications_cache"
	UsersNotificationsVulnerabilitiesPath = "/v1_users_notifications_vulnerability"
	AttackChainsPath                      = "/v1_attack_chains"
	RuntimeIncidentPath                   = "/v1_runtime_incident"
	RuntimeAlertPath                      = "/v1_runtime_alert"
	RuntimeIncidentPolicyPath             = "/v1_runtime_incident_policy"
	IntegrationReferencePath              = "/v1_integration_reference"
	CloudAccountPath                      = "/v1_cloud_account"

	//DB collections
	ClustersCollection                          = "clusters"
	PostureExceptionPolicyCollection            = "v1_posture_exception_policies"
	VulnerabilityExceptionPolicyCollection      = "v1_vulnerability_exception_policies"
	CustomerConfigCollection                    = "v1_customer_configurations"
	CustomersCollection                         = "customers"
	FrameworkCollection                         = "v1_opa_frameworks"
	RepositoryCollection                        = "v1_repositories"
	RegistryCronJobCollection                   = "v1_registry_cron_jobs"
	CollaborationConfigCollection               = "v1_collaboration_configurations"
	UsersNotificationsCacheCollection           = "v1_users_notifications_cache"
	UsersNotificationsVulnerabilitiesCollection = "v1_users_notifications_vulnerabilities"
	AttackChainsCollection                      = "v1_attack_chains"
	RuntimeIncidentCollection                   = "v1_runtime_incidents"
	RuntimeIncidentPolicyCollection             = "v1_runtime_incident_policies"
	IntegrationReferenceCollection              = "v1_integration_references"
	TokensCollection                            = "tokens"
	CloudAccountsCollection                     = "v1_cloud_accounts"

	//Common document fields
	IdField          = "_id"
	GUIDField        = "guid"
	NameField        = "name"
	AttributesField  = "attributes"
	CustomersField   = "customers"
	UpdatedTimeField = "updatedTime"
	//cluster fields
	ShortNameAttribute = "alias"
	ShortNameField     = AttributesField + "." + ShortNameAttribute

	//Query params
	ListParam          = "list"
	PolicyNameParam    = "policyName"
	FrameworkNameParam = "frameworkName"
	CustomersParam     = "customers"
	LimitParam         = "limit"
	SkipParam          = "skip"
	FromDateParam      = "fromDate"
	ToDateParam        = "toDate"
	ProjectionParam    = "projection"

	//Cached documents keys
	DefaultCustomerConfigKey = "defaultCustomerConfig"

	//customer configuration fields
	GlobalConfigName   = "default"
	CustomerConfigName = "CustomerConfig"
	ClusterNameParam   = "clusterName"
	ConfigNameParam    = "configName"
	ScopeParam         = "scope"
	CustomerScope      = "customer"
	DefaultScope       = "default"
)
