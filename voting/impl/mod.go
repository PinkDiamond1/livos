package impl

import (
	"fmt"

	"github.com/dedis/livos/storage"
	"github.com/dedis/livos/voting"
	"golang.org/x/xerrors"
)

//the voting implementation

const PERCENTAGE = 100

type VotingInstance struct {
	//voting instance's id
	Id string

	//parameters of the Voting
	Config voting.VotingConfig

	// open / closed
	Status string

	// Votes contains the choice of each voter, references by a userID
	Votes map[string]voting.Choice

	//db.socket personnalisé pour chacun ?
}

func (vi *VotingInstance) CastVote(user *voting.User) error {
	if vi.Status == "close" {
		return xerrors.Errorf("Impossible to cast the vote, the voting instance is closed.")
	}

	if val, ok := vi.Votes[user.UserID]; ok {
		for name, value := range user.MyChoice.VoteValue {
			additionOfLiquid, err := AddLiquid(val.VoteValue[name], value)
			if err != nil {
				return xerrors.Errorf(err.Error())
			}
			vi.Votes[user.UserID].VoteValue[name] = additionOfLiquid
		}
	} else {
		vi.Votes[user.UserID] = user.MyChoice
	}

	//update history of choice with the current choice
	histVoteValue := make(map[string]voting.Liquid)
	for key, value := range user.MyChoice.VoteValue {
		histVoteValue[key] = value
	}
	histChoice, err := NewChoice(histVoteValue)
	if err != nil {
		return xerrors.Errorf(err.Error())
	}
	user.HistoryOfChoice = append(user.HistoryOfChoice, histChoice)

	return nil
}

func (vi *VotingInstance) CloseVoting() {
	vi.SetStatus("close")
}

func (vi *VotingInstance) SetStatus(status string) error {
	if !(status == "open") && !(status == "close") {
		return xerrors.Errorf("The status is incorrect, should be either 'open' or 'close'.")
	}
	vi.Status = status
	return nil
}

func (vi *VotingInstance) GetStatus() string {
	return vi.Status
}

func (vi *VotingInstance) GetConfig() voting.VotingConfig {
	return vi.Config
}

//Give the result of the choices of the voting instance in the form: map[no:50 yes:50]
func (vi *VotingInstance) GetResults() map[string]float64 {
	fmt.Println("AFFICHAGE DES VOTES INDIVIDUELS : ")
	for s, v := range vi.Votes {
		fmt.Println(s, " votes: ", v)
	}

	results := make(map[string]float64, len(vi.Votes))
	counter := 0
	var yesPower float64 = 0
	var noPower float64 = 0
	for _, v := range vi.Votes {
		yesPower += v.VoteValue["yes"].Percentage
		noPower += v.VoteValue["no"].Percentage
		counter++
	}
	//in order to get 4 and not 4.6666666... for example
	// var temp1 = float64(int(yesPower/float64(counter))*100) / 100
	// var temp2 = float64(int(noPower/float64(counter))*100) / 100
	results["yes"] = yesPower / float64(len(vi.Config.Voters))
	results["no"] = noPower / float64(len(vi.Config.Voters))

	return results
}

func (vi *VotingInstance) GetUser(userID string) (*voting.User, error) {

	for _, x := range vi.Config.Voters {
		fmt.Println("config.voters elem = ", *x)
	}

	for _, value := range vi.Config.Voters {
		fmt.Println("value.UserID is : ", value.UserID)
		fmt.Println("userID is : ", userID)
		if value.UserID == userID {
			fmt.Println("value returned is = ", value)
			return value, nil
		}
	}
	return nil, xerrors.Errorf("Cannot find the user. UserId is %s", userID)
}

type VotingSystem struct {
	//contain all the votingInstances mapped to their stringID
	VotingInstancesList map[string]*VotingInstance

	//database
	Database storage.DB
}

//creation of a voting system, passing db and map as arguments
func NewVotingSystem(db storage.DB, vil map[string]*VotingInstance) VotingSystem {
	return VotingSystem{
		Database:            db,
		VotingInstancesList: vil,
	}
}

//creation of a voting instance
func (vs VotingSystem) CreateAndAdd(id string, config voting.VotingConfig, status string, votes map[string]voting.Choice) (*VotingInstance, error) {

	//check if id is null
	if id == "" {
		return nil, xerrors.Errorf("The id is empty.")
	}

	//check if status is open or close only
	if !(status == "open") && !(status == "close") {
		return nil, xerrors.Errorf("The status is incorrect, should be either 'open' or 'close'.")
	}

	//fmt.Println("Votes: ", votes)

	//create the object votingInstance
	var vi = VotingInstance{
		Id:     id,
		Config: config,
		Status: status,
		Votes:  votes,
	}

	p := &vi
	*p = vi

	//adding vi to the list of vi's of the voting system
	vs.VotingInstancesList[id] = p

	return p, nil
}

func (vs VotingSystem) Delete(id string) error {

	vi := vs.VotingInstancesList[id]
	if vi.Status == "open" {
		//vi.Status = "close"
		return xerrors.Errorf("Can't delete the votingInsance because it is still open")
	} else {
		delete(vs.VotingInstancesList, id)
	}
	return nil
}

//Return a list of all the voting instance
func (vs VotingSystem) ListVotings() []string {
	listeDeVotes := make([]string, len(vs.VotingInstancesList))
	for key := range vs.VotingInstancesList {
		if vs.VotingInstancesList[key].Status == "open" {
			listeDeVotes = append(listeDeVotes, key)
		}
	}
	return listeDeVotes
}

//Do we need to make a check to see if the id is null or letters or in fact
//doesn't belong to the list of ids
func (vs VotingSystem) GetVotingInstance(id string) VotingInstance {
	return *vs.VotingInstancesList[id]
}

func NewVotingConfig(voters []*voting.User, title string, desc string, cand []string) (voting.VotingConfig, error) {
	if title == "" {
		return voting.VotingConfig{}, xerrors.Errorf("title is empty")
	}

	return voting.VotingConfig{
		Voters:      voters,
		Title:       title,
		Description: desc,
		Candidates:  cand,
	}, nil
}

func (vs *VotingSystem) NewUser(userID string, delegTo map[string]voting.Liquid, delegFrom map[string]voting.Liquid, choice voting.Choice, historyOfChoice []voting.Choice) (voting.User, error) {

	// if votingPower > (float64(delegFrom)+1)*PERCENTAGE {
	// 	return voting.Choice{}, xerrors.Errorf("Voting power is too much : %f", votingPower)
	// }

	// //check that the sum overall votes is less or equal to the voting power
	// var sum float64 = 0
	// for _, value := range deleg {
	// 	sum += value.Percentage
	// }
	// for _, value := range choice {
	// 	sum += value.Percentage
	// }
	// if sum > (votingPower + float64(delegFrom)*PERCENTAGE) {
	// 	return voting.Choice{}, xerrors.Errorf("Cumulate voting power distributed is greater than the voting power. Was: %f, must not be greater thant %f.", sum, votingPower)
	// }

	return voting.User{
		UserID:          userID,
		DelegatedTo:     delegTo,
		DelegatedFrom:   delegFrom,
		MyChoice:        choice,
		VotingPower:     PERCENTAGE,
		HistoryOfChoice: historyOfChoice,
	}, nil
}

func NewChoice(voteValue map[string]voting.Liquid) (voting.Choice, error) {
	return voting.Choice{
		VoteValue: voteValue,
	}, nil
}

func NewLiquid(p float64) (voting.Liquid, error) {
	if p < 0 {
		return voting.Liquid{}, xerrors.Errorf("Init value is incorrect: Was %f, must be positive.", p)
	}

	return voting.Liquid{
		Percentage: p,
	}, nil
}

func AddLiquid(l1 voting.Liquid, l2 voting.Liquid) (voting.Liquid, error) {
	result, err := NewLiquid(l1.Percentage + l2.Percentage)
	if err != nil {
		return voting.Liquid{}, xerrors.Errorf(err.Error())
	}
	return result, err
}

func (vi *VotingInstance) CheckVotingPower(user *voting.User) error {
	if !(user.VotingPower >= 0) {
		return xerrors.Errorf("Value of voting power is negative: Was %f, must be more or equal than 0", user.VotingPower)
	}
	return nil
}

func (vi *VotingInstance) CheckVotingPowerOfVoters() bool {
	for _, user := range vi.Config.Voters {
		if user.VotingPower != 0. {
			return true
		}
	}
	return false
}

func (vi *VotingInstance) SetChoice(user *voting.User, choice voting.Choice) error {

	var sumOfVotingPower float64 = 0
	for _, v := range choice.VoteValue {
		sumOfVotingPower += v.Percentage
	}

	if user.VotingPower-sumOfVotingPower >= 0 {
		user.VotingPower -= sumOfVotingPower
	}

	err := vi.CheckVotingPower(user)
	if err != nil {
		return xerrors.Errorf(err.Error())
	}

	//update the current choice with the new one
	user.MyChoice = choice

	return nil
}

func (vi *VotingInstance) DelegTo(userSend *voting.User, userReceive *voting.User, quantity voting.Liquid) error {

	//CANNOT DELEGATE 0 voting power, do nothing if it is the case
	if quantity.Percentage > 0 {
		new_quantity, err := AddLiquid(quantity, userReceive.DelegatedFrom[userSend.UserID])
		if err != nil {
			return xerrors.Errorf(err.Error())
		}
		userReceive.DelegatedFrom[userSend.UserID] = new_quantity

		userSend.DelegatedTo[userReceive.UserID] = new_quantity

		userReceive.VotingPower += quantity.Percentage
		err = vi.CheckVotingPower(userReceive)
		if err != nil {
			return xerrors.Errorf(err.Error())
		}

		userSend.VotingPower -= quantity.Percentage
		err = vi.CheckVotingPower(userSend)
		if err != nil {
			return xerrors.Errorf(err.Error())
		}
	}

	return nil
}
