package boards

type CreateBoardRequest struct {
	Name  string `json:"name" binding:"required"`
	Owner string `json:"owner" binding:"required"`
}

type UpdateBoardNameRequest struct {
	Name string `json:"name" binding:"required"`
}

type CollaboratorRequest struct {
	CognitoID string `json:"cognito_id" binding:"required"`
}
