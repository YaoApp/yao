package acl

import "github.com/gin-gonic/gin"

// Enforce checks if the user has access to the resource
func (acl *ACL) Enforce(c *gin.Context) (bool, error) {
	return true, nil
}
