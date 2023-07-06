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

	//PATHS
	ClusterPath                      = "/cluster"
	PostureExceptionPolicyPath       = "/v1_posture_exception_policy"
	VulnerabilityExceptionPolicyPath = "/v1_vulnerability_exception_policy"
	CustomerConfigPath               = "/v1_customer_configuration"
	FrameworkPath                    = "/v1_opa_framework"
	RepositoryPath                   = "/v1_repository"
	AdminPath                        = "/v1_admin"
	CustomerPath                     = "/customer"
	TenantPath                       = "/customer_tenant"
	RegistryCronJobPath              = "/v1_registry_cron_job"
	NotificationConfigPath           = "/v1_notification_config"
	CustomerStatePath                = "/v1_customer_state"
	ActiveSubscriptionPath           = "/v1_active_subscription"
	CollaborationConfigPath          = "/v1_collaboration_configuration"

	//DB collections
	ClustersCollection                     = "clusters"
	PostureExceptionPolicyCollection       = "v1_posture_exception_policies"
	VulnerabilityExceptionPolicyCollection = "v1_vulnerability_exception_policies"
	CustomerConfigCollection               = "v1_customer_configurations"
	CustomersCollection                    = "customers"
	FrameworkCollection                    = "v1_opa_frameworks"
	RepositoryCollection                   = "v1_repositories"
	RegistryCronJobCollection              = "v1_registry_cron_jobs"
	CollaborationConfigCollection          = "v1_collaboration_configurations"

	//Common document fields
	IdField          = "_id"
	GUIDField        = "guid"
	NameField        = "name"
	DeletedField     = "is_deleted"
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
