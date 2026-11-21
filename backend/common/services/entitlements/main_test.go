package entitlements

import (
	"testing"
	"time"

	"github.com/Secreto31126/tesis/common/mocks"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

type counts map[string]int64

func (c counts) CountOwnedOrgs(u string) (int64, error)       { return c["orgs:"+u], nil }
func (c counts) CountOwnedGroups(u string) (int64, error)     { return c["groups:"+u], nil }
func (c counts) CountSecretsBilledTo(u string) (int64, error) { return c["secrets:"+u], nil }
func (c counts) CountPostitTemplates(s models.PostitTemplateScope) (int64, error) {
	return c["templates:"+s.Owner], nil
}

var now = time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)

func fixture(users []models.User, orgs []models.Org, c counts) *Service {
	return New(&mocks.MemoryUserStore{Users: users}, &mocks.MemoryOrgStore{Orgs: orgs}, &mocks.MockDB{}, &mocks.MemoryGroupStore{}, c, nil, func() time.Time { return now })
}

func TestForUser_PlansExpiryAndUnknownAccounts(t *testing.T) {
	past, future := now.Add(-time.Hour), now.Add(time.Hour)
	pro := models.User{Id: uuid.New(), Plan: models.PlanPro, PlanExpires: &future}
	lapsed := models.User{Id: uuid.New(), Plan: models.PlanMax, PlanExpires: &past}
	srv := fixture([]models.User{pro, lapsed}, nil, nil)

	for _, c := range []struct {
		id   string
		want models.Plan
	}{{pro.Id.String(), models.PlanPro}, {lapsed.Id.String(), models.PlanFree}, {uuid.NewString(), models.PlanFree}, {"not-a-uuid", models.PlanFree}} {
		got, err := srv.ForUser(c.id)
		if err != nil || got.Plan != c.want || got.Limits != models.LimitsFor(c.want) {
			t.Errorf("ForUser(%s) = %+v, %v; want %s", c.id, got, err, c.want)
		}
	}
}

func TestForBoard_OrgBoardsRunOnThePayersPlan(t *testing.T) {
	payer := models.User{Id: uuid.New(), Plan: models.PlanMax}
	owner := models.User{Id: uuid.New()}
	org := models.Org{Id: uuid.New(), Owner: payer.Id.String()}
	srv := fixture([]models.User{payer, owner}, []models.Org{org}, nil)

	got, err := srv.ForBoard(&models.Board{Owner: owner.Id.String(), Org: &org.Id})
	if err != nil || got.Plan != models.PlanMax || got.Payer != payer.Id.String() {
		t.Errorf("org board = %+v, %v", got, err)
	}
	got, _ = srv.ForBoard(&models.Board{Owner: owner.Id.String()})
	if got.Plan != models.PlanFree || got.Payer != owner.Id.String() {
		t.Errorf("personal board = %+v", got)
	}
	gone := uuid.New()
	got, _ = srv.ForBoard(&models.Board{Owner: owner.Id.String(), Org: &gone})
	if got.Payer != owner.Id.String() {
		t.Errorf("a deleted org falls back to the owner: %+v", got)
	}
}

func TestForOrg_LegacyOrgsUseTheirFirstAdmin(t *testing.T) {
	admin := models.User{Id: uuid.New(), Plan: models.PlanPro}
	org := &models.Org{Id: uuid.New(), Members: []models.OrgMember{{User: "m", Role: models.OrgRoleMember}, {User: admin.Id.String(), Role: models.OrgRoleAdmin}}}
	got, _ := fixture([]models.User{admin}, nil, nil).ForOrg(org)
	if got.Plan != models.PlanPro {
		t.Errorf("plan = %s", got.Plan)
	}
}

func TestSecretsLeftAndUsage(t *testing.T) {
	u := models.User{Id: uuid.New(), Plan: models.PlanPro}
	id := u.Id.String()
	srv := fixture([]models.User{u}, nil, counts{"secrets:" + id: 30, "orgs:" + id: 1, "groups:" + id: 2, "templates:" + id: 3})

	payer, _ := srv.ForUser(id)
	if left, _ := srv.SecretsLeft(payer); left != 0 {
		t.Errorf("over the limit still left %d", left)
	}
	usage, err := srv.Usage(id)
	if err != nil || usage != (models.Usage{Orgs: 1, Groups: 2, Secrets: 30, UserTemplates: 3}) {
		t.Errorf("usage = %+v, %v", usage, err)
	}
}

func TestForSecretScope_SystemHasNoPayer(t *testing.T) {
	got, err := fixture(nil, nil, nil).ForSecretScope(models.SystemScope)
	if err != nil || got.Payer != "" {
		t.Errorf("system = %+v, %v", got, err)
	}
}

func TestForBoard_OrgWithoutPayerFallsBackToTheBoardOwner(t *testing.T) {
	owner := models.User{Id: uuid.New(), Plan: models.PlanPro}
	org := models.Org{Id: uuid.New()}
	srv := fixture([]models.User{owner}, []models.Org{org}, nil)

	got, err := srv.ForBoard(&models.Board{Owner: owner.Id.String(), Org: &org.Id})
	if err != nil || got.Payer != owner.Id.String() || got.Plan != models.PlanPro {
		t.Errorf("board = %+v, %v", got, err)
	}

	id := uuid.New()
	got, err = srv.ForSecretScope(models.SecretScope{Kind: models.ScopeBoard, Owner: id.String()})
	if err != nil || got.Payer != "" {
		t.Errorf("unknown board scope = %+v, %v", got, err)
	}
}
