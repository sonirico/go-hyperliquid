package hyperliquid

//go:generate easyjson -all

// UserRateLimit is the response of the userRateLimit info request.
type UserRateLimit struct {
	// Cumulative traded volume, which determines the request cap.
	CumVlm string `json:"cumVlm"`
	// max(0, cumulative requests used minus reserved requests)
	NRequestsUsed int64 `json:"nRequestsUsed"`
	NRequestsCap  int64 `json:"nRequestsCap"`
	// max(0, reserved requests minus cumulative requests used)
	NRequestsSurplus int64 `json:"nRequestsSurplus"`
}

// UserRoleType is the role of an address, as returned by the userRole info request.
type UserRoleType string

const (
	UserRoleMissing    UserRoleType = "missing"
	UserRoleUser       UserRoleType = "user"
	UserRoleAgent      UserRoleType = "agent"
	UserRoleVault      UserRoleType = "vault"
	UserRoleSubAccount UserRoleType = "subAccount"
)

// UserRole is the response of the userRole info request. Data is only set for
// agents (the user the agent trades for) and sub-accounts (their master account).
type UserRole struct {
	Role UserRoleType  `json:"role"`
	Data *UserRoleData `json:"data,omitempty"`
}

type UserRoleData struct {
	User   string `json:"user,omitempty"`
	Master string `json:"master,omitempty"`
}

// PredictedFunding is the predicted funding of one coin on one venue.
type PredictedFunding struct {
	FundingRate     string `json:"fundingRate"`
	NextFundingTime int64  `json:"nextFundingTime"`
	// Funding interval in hours. Not always present.
	FundingIntervalHours *int `json:"fundingIntervalHours,omitempty"`
}

// VenuePredictedFunding is the predicted funding on one venue, e.g. "HlPerp",
// "BinPerp" or "BybitPerp". Funding is nil when the venue has no data for the coin.
type VenuePredictedFunding struct {
	Venue   string
	Funding *PredictedFunding
}

// CoinPredictedFundings groups the predicted fundings of one coin by venue.
type CoinPredictedFundings struct {
	Coin   string
	Venues []VenuePredictedFunding
}
