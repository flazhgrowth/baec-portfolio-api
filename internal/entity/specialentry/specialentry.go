package specialentry

type (
	CMSRegisterRequest struct {
		MaxEntry int `json:"max_entry"`
	}
	CMSRegisterResponse struct {
		Token string `json:"token"`
	}
)

type (
	ValidateRequest struct {
		Token string `json:"token"`
	}
)
