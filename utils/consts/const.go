package consts

const (

	//Context keys
	DocContentKey = "docContent"
	CustomerGUID  = "customerGUID"
	Collection    = "collection"
	ReqLogger     = "reqLogger"
	AdminAccess   = "adminAccess"

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

	//DB collections
	ClustersCollection                     = "clusters"
	PostureExceptionPolicyCollection       = "v1_posture_exception_policies"
	VulnerabilityExceptionPolicyCollection = "v1_vulnerability_exception_policies"
	CustomerConfigCollection               = "v1_customer_configurations"
	CustomersCollection                    = "customers"
	FrameworkCollection                    = "v1_opa_frameworks"
	RepositoryCollection                   = "v1_repositories"

	//Common document fields
	IdField         = "_id"
	GUIDField       = "guid"
	NameField       = "name"
	DeletedField    = "is_deleted"
	AttributesField = "attributes"
	CustomersField  = "customers"
	//cluster fields
	ShortNameAttribute = "alias"
	ShortNameField     = AttributesField + "." + ShortNameAttribute

	//Query params
	ListParam          = "list"
	PolicyNameParam    = "policyName"
	FrameworkNameParam = "frameworkName"
	CustomersParam     = "customers"

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