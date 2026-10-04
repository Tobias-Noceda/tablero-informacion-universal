package groups

type CreateGroupRequest struct {
	Name string `json:"name" binding:"required"`
}

type MemberRequest struct {
	Member string `json:"member" binding:"required"`
}
