package rm

import (
	"fmt"
	"sync"

	"github.com/pkg/errors"

	"github.com/seata/seata-go/pkg/common/log"
)

const (
	RemotingUnknow  = 0
	RemotingGrpc    = 1
	RemotingDubbogo = 2
	RemotingLocal   = 3
)

type RemotingParser interface {
	IsService(target interface{}) bool
	IsReference(target interface{}) bool
	IsRemoting(target interface{}) bool
	ParseService(target interface{}) (*TwoPhaseAction, error)
	ParseReference(target interface{}) (*TwoPhaseAction, error)
	ParseTwoPhaseActionByInterface(target interface{}) (*TwoPhaseAction, error)
	GetRemotingType(target interface{}) int
}

func RegisterRemotingParse(parse RemotingParser) {
	remotingParseTable = append(remotingParseTable, parse)
}

type DefaultRemotingParser struct{}

var (
	once                           sync.Once
	remotingParseTable             = make([]RemotingParser, 0)
	defaultRemotingParserSingleton *DefaultRemotingParser
)

func GetDefaultRemotingParser() *DefaultRemotingParser {
	if defaultRemotingParserSingleton == nil {
		once.Do(func() {
			defaultRemotingParserSingleton = &DefaultRemotingParser{}
		})
	}
	return defaultRemotingParserSingleton

}

func (d *DefaultRemotingParser) IsService(target interface{}) bool {
	for _, v := range remotingParseTable {
		if v.IsService(target) {
			return true
		}
	}
	return false
}

func (d *DefaultRemotingParser) IsReference(target interface{}) bool {
	for _, v := range remotingParseTable {
		if v.IsReference(target) {
			return true
		}
	}
	return false
}

func (d *DefaultRemotingParser) IsRemoting(target interface{}) bool {
	for _, v := range remotingParseTable {
		if v.IsRemoting(target) {
			return true
		}
	}
	return false
}

func (d *DefaultRemotingParser) ParseService(target interface{}) (*TwoPhaseAction, error) {
	for _, v := range remotingParseTable {
		if res, err := v.ParseService(target); err != nil {
			log.Warnf("parse service failed : %s", err)
		} else if res != nil {
			return res, nil
		}
	}
	return nil, errors.New(fmt.Sprintf("don't find a parser to resolve service %v", target))
}

func (d *DefaultRemotingParser) ParseReference(target interface{}) (*TwoPhaseAction, error) {
	for _, v := range remotingParseTable {
		if res, err := v.ParseReference(target); err != nil {
			log.Warnf("parse reference failed : %s", err)
		} else if res != nil {
			return res, nil
		}
	}
	return nil, errors.New(fmt.Sprintf("don't find a parser to resolve reference %v", target))
}

func (d *DefaultRemotingParser) ParseTwoPhaseActionByInterface(target interface{}) (*TwoPhaseAction, error) {
	for _, v := range remotingParseTable {
		if res, err := v.ParseTwoPhaseActionByInterface(target); err != nil {
			return nil, err
		} else if res != nil {
			return res, nil
		}
	}

	return nil, errors.New(fmt.Sprintf("don't find a parser to resolve two phase action %v", target))
}

func (d *DefaultRemotingParser) GetRemotingType(target interface{}) int {
	for _, v := range remotingParseTable {
		if res := v.GetRemotingType(target); res != RemotingUnknow {
			return res
		}
	}
	return RemotingUnknow
}
