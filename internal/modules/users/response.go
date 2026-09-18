package users

// Role adalah role RBAC ringkas milik user dalam satu organisasi.
type Role struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// OrgUser adalah response user pada endpoint org-scoped, dilengkapi
// daftar role RBAC user tersebut dalam organisasi terkait.
type OrgUser struct {
	User
	Roles []Role `json:"roles"`
}
