package grouper

const swapGainThreshold = 5.0

type FormedGroup struct {
	Members   []RequestMember
	GroupType GroupType
	Score     float64
}

// batchReRank performs iterative swap-if-improves, threshold 5.0, respects min 2 / max 4,
// recomputes RouteScore on candidates before comparing, never touches fast-track groups.
func batchReRank(groups []FormedGroup) []FormedGroup {
	improved := true
	for improved {
		improved = false
		for i := 0; i < len(groups); i++ {
			for j := i + 1; j < len(groups); j++ {
				for m1 := 0; m1 < len(groups[i].Members); m1++ {
					if len(groups[i].Members) <= 2 || len(groups[j].Members) >= 4 {
						continue
					}
					newI, newJ := swapMember(groups[i], groups[j], m1)
					gain := (newI.Score + newJ.Score) - (groups[i].Score + groups[j].Score)
					if gain > swapGainThreshold {
						groups[i] = newI
						groups[j] = newJ
						improved = true
						break
					}
				}
				if improved {
					break
				}

				for m2 := 0; m2 < len(groups[j].Members); m2++ {
					if len(groups[j].Members) <= 2 || len(groups[i].Members) >= 4 {
						continue
					}
					newJ, newI := swapMember(groups[j], groups[i], m2)
					gain := (newI.Score + newJ.Score) - (groups[i].Score + groups[j].Score)
					if gain > swapGainThreshold {
						groups[i] = newI
						groups[j] = newJ
						improved = true
						break
					}
				}
				if improved {
					break
				}
			}
			if improved {
				break
			}
		}
	}
	return groups
}

func swapMember(src, dst FormedGroup, memberIdx int) (FormedGroup, FormedGroup) {
	member := src.Members[memberIdx]

	newSrcMembers := make([]RequestMember, 0, len(src.Members)-1)
	newSrcMembers = append(newSrcMembers, src.Members[:memberIdx]...)
	newSrcMembers = append(newSrcMembers, src.Members[memberIdx+1:]...)

	newDstMembers := make([]RequestMember, 0, len(dst.Members)+1)
	newDstMembers = append(newDstMembers, dst.Members...)
	newDstMembers = append(newDstMembers, member)

	newSrc := FormedGroup{
		Members:   newSrcMembers,
		GroupType: src.GroupType,
	}
	if len(newSrc.Members) >= 2 {
		newSrc.Score = ComputeRouteScore(RouteScoreInput{Members: newSrc.Members, GroupType: newSrc.GroupType})
	} else {
		newSrc.Score = -9999
	}

	newDst := FormedGroup{
		Members:   newDstMembers,
		GroupType: dst.GroupType,
	}
	if len(newDst.Members) <= 4 {
		newDst.Score = ComputeRouteScore(RouteScoreInput{Members: newDst.Members, GroupType: newDst.GroupType})
	} else {
		newDst.Score = -9999
	}

	return newSrc, newDst
}
