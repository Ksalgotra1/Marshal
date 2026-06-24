package grouper

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func createMember(id string, lat, lng float64) RequestMember {
	return RequestMember{
		StudentID:  id,
		PickupLat:  lat,
		PickupLng:  lng,
		DropoffLat: lat + 0.01,
		DropoffLng: lng + 0.01,
		ArriveBy:   time.Now().Add(1 * time.Hour),
	}
}

func scoreGroup(members []RequestMember) float64 {
	return ComputeRouteScore(RouteScoreInput{Members: members, GroupType: GroupTypeExact})
}

func TestReRankSwapImproves(t *testing.T) {
	// North cluster
	m1 := createMember("N1", 30.5, 76.5)
	m2 := createMember("N2", 30.501, 76.501)
	m3 := createMember("N3", 30.502, 76.502)

	// South cluster
	m4 := createMember("S1", 20.0, 70.0)
	m5 := createMember("S2", 20.001, 70.001)

	// group 1 has N1, N2. group 2 has S1, S2, and N3 (outlier)
	g1 := FormedGroup{Members: []RequestMember{m1, m2}, GroupType: GroupTypeExact}
	g1.Score = scoreGroup(g1.Members)

	g2 := FormedGroup{Members: []RequestMember{m4, m5, m3}, GroupType: GroupTypeExact}
	g2.Score = scoreGroup(g2.Members)

	groups := []FormedGroup{g1, g2}
	reranked := batchReRank(groups)

	// N3 should move to g1
	assert.Len(t, reranked[0].Members, 3)
	assert.Len(t, reranked[1].Members, 2)

	// Check total score improved
	oldTotal := g1.Score + g2.Score
	newTotal := reranked[0].Score + reranked[1].Score
	assert.Greater(t, newTotal-oldTotal, swapGainThreshold)
}

func TestReRankNoSwapBelowThreshold(t *testing.T) {
	// Same clusters, but they are already optimal
	m1 := createMember("N1", 30.5, 76.5)
	m2 := createMember("N2", 30.501, 76.501)
	m3 := createMember("N3", 30.502, 76.502)

	m4 := createMember("S1", 20.0, 70.0)
	m5 := createMember("S2", 20.001, 70.001)

	g1 := FormedGroup{Members: []RequestMember{m1, m2, m3}, GroupType: GroupTypeExact}
	g1.Score = scoreGroup(g1.Members)

	g2 := FormedGroup{Members: []RequestMember{m4, m5}, GroupType: GroupTypeExact}
	g2.Score = scoreGroup(g2.Members)

	groups := []FormedGroup{g1, g2}
	reranked := batchReRank(groups)

	// Should not change
	assert.Len(t, reranked[0].Members, 3)
	assert.Len(t, reranked[1].Members, 2)
	assert.Equal(t, "N1", reranked[0].Members[0].StudentID)
}

func TestReRankSwapRespectMinSize(t *testing.T) {
	// g2 has an outlier that belongs in g1, but g2 only has 2 members.
	// Swapping it would reduce g2 to 1 member, violating min size 2.
	m1 := createMember("N1", 30.5, 76.5)
	m2 := createMember("N2", 30.501, 76.501)

	m3 := createMember("S1", 20.0, 70.0)
	m4 := createMember("N3", 30.502, 76.502) // belongs in g1!

	g1 := FormedGroup{Members: []RequestMember{m1, m2}, GroupType: GroupTypeExact}
	g1.Score = scoreGroup(g1.Members)

	g2 := FormedGroup{Members: []RequestMember{m3, m4}, GroupType: GroupTypeExact}
	g2.Score = scoreGroup(g2.Members)

	groups := []FormedGroup{g1, g2}
	reranked := batchReRank(groups)

	// Should not change because g2 would drop to 1
	assert.Len(t, reranked[0].Members, 2)
	assert.Len(t, reranked[1].Members, 2)
}

func TestReRankSwapRespectMaxSize(t *testing.T) {
	// g1 already has 4 members. g2 has an outlier that belongs in g1.
	// Swapping it would push g1 to 5 members, violating max size 4.
	m1 := createMember("N1", 30.5, 76.5)
	m2 := createMember("N2", 30.501, 76.501)
	m3 := createMember("N3", 30.502, 76.502)
	m4 := createMember("N4", 30.503, 76.503)

	m5 := createMember("S1", 20.0, 70.0)
	m6 := createMember("S2", 20.001, 70.001)
	m7 := createMember("N5", 30.504, 76.504) // belongs in g1

	g1 := FormedGroup{Members: []RequestMember{m1, m2, m3, m4}, GroupType: GroupTypeExact}
	g1.Score = scoreGroup(g1.Members)

	g2 := FormedGroup{Members: []RequestMember{m5, m6, m7}, GroupType: GroupTypeExact}
	g2.Score = scoreGroup(g2.Members)

	groups := []FormedGroup{g1, g2}
	reranked := batchReRank(groups)

	// Should not change because g1 would grow to 5
	assert.Len(t, reranked[0].Members, 4)
	assert.Len(t, reranked[1].Members, 3)
}

func TestReRankIdempotent(t *testing.T) {
	m1 := createMember("N1", 30.5, 76.5)
	m2 := createMember("N2", 30.501, 76.501)
	m3 := createMember("N3", 30.502, 76.502)

	m4 := createMember("S1", 20.0, 70.0)
	m5 := createMember("S2", 20.001, 70.001)

	g1 := FormedGroup{Members: []RequestMember{m1, m2}, GroupType: GroupTypeExact}
	g1.Score = scoreGroup(g1.Members)

	g2 := FormedGroup{Members: []RequestMember{m4, m5, m3}, GroupType: GroupTypeExact}
	g2.Score = scoreGroup(g2.Members)

	groups := []FormedGroup{g1, g2}
	run1 := batchReRank(groups)

	// deep copy to avoid slice modifications
	run1Copy := make([]FormedGroup, len(run1))
	copy(run1Copy, run1)

	run2 := batchReRank(run1Copy)

	// Both should be identical
	assert.Equal(t, run1, run2)
}
