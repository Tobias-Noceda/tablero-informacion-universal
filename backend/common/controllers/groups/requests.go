package groups

type CreateGroupRequest struct {
	CognitoID string `json:"cognito_id" binding:"required"`
	Name      string `json:"name" binding:"required"`
}

type MemberRequest struct {
	CognitoID string `json:"cognito_id" binding:"required"`
	Member    string `json:"member" binding:"required"`
}
