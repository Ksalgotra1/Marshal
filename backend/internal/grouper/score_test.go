package grouper

import (
	"testing"
	"time"
)

func TestTightClusterHighScore(t *testing.T) {
	now := time.Now()
	m1 := RequestMember{"A", 30.0, 76.0, 30.5, 76.5, now}
	m2 := RequestMember{"B", 30.001, 76.001, 30.5, 76.5, now.Add(2 * time.Minute)}
	m3 := RequestMember{"C", 30.002, 76.002, 30.5, 76.5, now.Add(4 * time.Minute)}

	in := RouteScoreInput{
		Members:   []RequestMember{m1, m2, m3},
		GroupType: GroupTypeExact,
	}
	score := ComputeRouteScore(in)
	if score <= 25 {
		t.Errorf("expected score > 25, got %f", score)
	}
}

func TestHighDetourLowScore(t *testing.T) {
	now := time.Now()
	// Hand-tuned coordinates to simulate 3 members 3km apart with 6+ km detour
	m1 := RequestMember{"A", 30.0, 76.0, 30.5, 76.0, now}
	m2 := RequestMember{"B", 30.0, 76.027, 30.5, 76.027, now}
	m3 := RequestMember{"C", 30.0, 76.054, 30.5, 76.054, now}

	in := RouteScoreInput{
		Members:   []RequestMember{m1, m2, m3},
		GroupType: GroupTypeExact,
	}
	score := ComputeRouteScore(in)
	if score >= 15 {
		t.Errorf("expected score < 15, got %f", score)
	}
}

func TestEnRouteLowerThanExact(t *testing.T) {
	now := time.Now()
	m1 := RequestMember{"A", 30.0, 76.0, 30.5, 76.5, now}
	m2 := RequestMember{"B", 30.1, 76.1, 30.4, 76.4, now}
	m3 := RequestMember{"C", 30.2, 76.2, 30.3, 76.3, now}

	exact := ComputeRouteScore(RouteScoreInput{
		Members:   []RequestMember{m1, m2, m3},
		GroupType: GroupTypeExact,
	})
	enroute := ComputeRouteScore(RouteScoreInput{
		Members:   []RequestMember{m1, m2, m3},
		GroupType: GroupTypeEnRoute,
	})
	if enroute >= exact {
		t.Errorf("expected en-route score (%f) to be lower than exact (%f)", enroute, exact)
	}
}

func TestCapacityBonusErasedByDetour(t *testing.T) {
	now := time.Now()
	// tight 3 member
	m1 := RequestMember{"A", 30.0, 76.0, 30.5, 76.5, now}
	m2 := RequestMember{"B", 30.0, 76.0, 30.5, 76.5, now}
	m3 := RequestMember{"C", 30.0, 76.0, 30.5, 76.5, now}
	tight := ComputeRouteScore(RouteScoreInput{
		Members:   []RequestMember{m1, m2, m3},
		GroupType: GroupTypeExact,
	})

	// 4 member with big detour
	m4_1 := RequestMember{"A", 30.0, 76.0, 30.5, 76.5, now}
	m4_2 := RequestMember{"B", 30.1, 76.0, 30.5, 76.5, now}
	m4_3 := RequestMember{"C", 30.2, 76.0, 30.5, 76.5, now}
	m4_4 := RequestMember{"D", 30.3, 76.0, 30.5, 76.5, now}
	spread := ComputeRouteScore(RouteScoreInput{
		Members:   []RequestMember{m4_1, m4_2, m4_3, m4_4},
		GroupType: GroupTypeExact,
	})

	if spread >= tight {
		t.Errorf("expected 4-member with detour (%f) to score lower than tight 3-member (%f)", spread, tight)
	}
}
