package assigner

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type AssignerUnitSuite struct {
	suite.Suite
}

func TestAssignerUnitSuite(t *testing.T) {
	suite.Run(t, new(AssignerUnitSuite))
}

func (s *AssignerUnitSuite) TestDynamicTimeoutIsClamped() {
	s.Equal(10*time.Minute, DynamicTimeout(time.Now().Add(8*time.Hour)))
	s.Equal(6*time.Minute, DynamicTimeout(time.Now().Add(time.Hour)))
	s.Equal(3*time.Minute, DynamicTimeout(time.Now().Add(30*time.Minute)))
	s.Equal(2*time.Minute, DynamicTimeout(time.Now().Add(5*time.Minute)))
}
