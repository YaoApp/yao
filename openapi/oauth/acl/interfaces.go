package acl

import "github.com/gin-gonic/gin"

// Enforcer interface is used to enforce access control rules
type Enforcer interface {
	// Enforce checks if a user has access to a resource
	Enforce(c *gin.Context) (bool, error)

	// Enabled returns true if the ACL is enabled, otherwise false
	Enabled() bool
}
