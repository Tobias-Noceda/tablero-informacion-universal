package orgs

import "github.com/Secreto31126/tesis/common/models"

type CreateOrgRequest struct {
	Name string `json:"name" binding:"required"`
}

type RenameOrgRequest struct {
	Name string `json:"name" binding:"required"`
}

type SetMemberRequest struct {
	Email string         `json:"email" binding:"required"`
	Role  models.OrgRole `json:"role" binding:"required"`
}
