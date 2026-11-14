package models

// Plan is what a user subscribed to. It belongs to the user; whatever a
// board, an organization or a group may hold comes from the plan of whoever
// pays for it.
type Plan string

const (
	PlanFree Plan = "free"
	PlanPro  Plan = "pro"
	PlanMax  Plan = "max"
)

// Plans are the plans on offer, cheapest first.
var Plans = []Plan{PlanFree, PlanPro, PlanMax}

var planRank = map[Plan]int{PlanFree: 1, PlanPro: 2, PlanMax: 3}

func (p Plan) Valid() bool {
	return planRank[p] > 0
}

// Rank orders the plans: a higher one allows everything a lower one does.
// An unknown plan ranks below all of them.
func (p Plan) Rank() int {
	return planRank[p]
}

// Limits is what a plan allows. Seats counts the people on a board, in an
// organization or in a group, its owner included, so 1 means no sharing.
// MinRate is the shortest time, in seconds, a card's result stays cached.
type Limits struct {
	Orgs           int `json:"orgs"`
	BoardsPerSpace int `json:"boards_per_space"`
	Seats          int `json:"seats"`
	// ViewersOnly: everyone but the owner joins a board as a viewer.
	ViewersOnly    bool `json:"viewers_only"`
	NodesPerBoard  int  `json:"nodes_per_board"`
	UserTemplates  int  `json:"user_templates"`
	BoardTemplates int  `json:"board_templates"`
	OrgTemplates   int  `json:"org_templates"`
	Secrets        int  `json:"secrets"`
	Groups         int  `json:"groups"`
	MinRate        int  `json:"min_rate"`
	ForceRefresh   bool `json:"force_refresh"`
	APIAccess      bool `json:"api_access"`
	PriceUSD       int  `json:"price_usd"`
}

var planLimits = map[Plan]Limits{
	PlanFree: {
		Orgs: 0, BoardsPerSpace: 5, Seats: 5, ViewersOnly: true, NodesPerBoard: 25,
		UserTemplates: 5, BoardTemplates: 0, OrgTemplates: 0,
		Secrets: 5, Groups: 0, MinRate: 600,
		ForceRefresh: false, APIAccess: false, PriceUSD: 0,
	},
	PlanPro: {
		Orgs: 3, BoardsPerSpace: 10, Seats: 5, NodesPerBoard: 100,
		UserTemplates: 10, BoardTemplates: 5, OrgTemplates: 3,
		Secrets: 25, Groups: 5, MinRate: 150,
		ForceRefresh: false, APIAccess: true, PriceUSD: 9,
	},
	PlanMax: {
		Orgs: 10, BoardsPerSpace: 50, Seats: 25, NodesPerBoard: 300,
		UserTemplates: 50, BoardTemplates: 50, OrgTemplates: 50,
		Secrets: 200, Groups: 25, MinRate: 60,
		ForceRefresh: true, APIAccess: true, PriceUSD: 29,
	},
}

// LimitsFor is the one table of what each plan allows, also served to the
// frontend. An unknown plan gets the free limits.
func LimitsFor(plan Plan) Limits {
	if limits, ok := planLimits[plan]; ok {
		return limits
	}
	return planLimits[PlanFree]
}

// The limits a request can run into, as the API names them.
const (
	LimitOrgs           = "orgs"
	LimitBoards         = "boards"
	LimitSeats          = "seats"
	LimitEditors        = "editors"
	LimitNodes          = "nodes"
	LimitUserTemplates  = "user_templates"
	LimitBoardTemplates = "board_templates"
	LimitOrgTemplates   = "org_templates"
	LimitSecrets        = "secrets"
	LimitGroups         = "groups"
	LimitForceRefresh   = "force_refresh"
	LimitAPIAccess      = "api_access"
)

// LimitError refuses something the payer's plan does not allow: one more
// than Max of Limit, or a feature the plan lacks (Max 0).
type LimitError struct {
	Limit string
	Max   int
	Plan  Plan
}

func (e *LimitError) Error() string {
	return "plan_limit"
}

// Entitlement is a plan as it applies to something: who pays for it and
// what their plan allows right now.
type Entitlement struct {
	// Payer is the user whose plan this is; empty when nobody pays (the
	// platform's own scope).
	Payer  string
	Plan   Plan
	Limits Limits
}

// EntitlementOf is plan as held by payer.
func EntitlementOf(payer string, plan Plan) Entitlement {
	return Entitlement{Payer: payer, Plan: plan, Limits: LimitsFor(plan)}
}

// Refuse is the error for running into limit, whose ceiling is max.
func (e Entitlement) Refuse(limit string, max int) *LimitError {
	return &LimitError{Limit: limit, Max: max, Plan: e.Plan}
}

// Admits fails once used has reached max: only one more is refused, what is
// already there stays.
func (e Entitlement) Admits(limit string, used, max int) error {
	if used >= max {
		return e.Refuse(limit, max)
	}
	return nil
}

// Usage is how much of their plan a user has taken, for the limits that are
// counted per user rather than per board.
type Usage struct {
	Orgs          int `json:"orgs"`
	Groups        int `json:"groups"`
	Secrets       int `json:"secrets"`
	UserTemplates int `json:"user_templates"`
}
