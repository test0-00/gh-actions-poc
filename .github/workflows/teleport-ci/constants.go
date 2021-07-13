package teleportci

const (

	// ASSIGN is the argument to assign reviewers
	ASSIGN = "assign-reviewers"
	// CHECK is the argument to check reviewers
	CHECK = "check-reviewers"
	// GITHUBEVENTPATH is the envvariable name that
	// has the path to the event payload json file
	GITHUBEVENTPATH = "GITHUB_EVENT_PATH"
)

const (
	APPROVED = "APPROVED"
	// ASSIGNMENTS is the environment variable name that stores
	// which reviewers should be assigned to which authors
	ASSIGNMENTS = "ASSIGNMENTS"
	// TOKEN is the env variable name that stores the Github authentication token
	TOKEN = "GITHUB_TOKEN"
)
