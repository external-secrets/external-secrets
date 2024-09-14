// Copyright External Secrets Inc. All Rights Reserved
package fake

import (
	"errors"
	"fmt"
	"math"

	"github.com/go-chef/chef"
)

const (
	CORRECTUSER = "correctUser"
	testitem    = "item03"
	DatabagName = "databag01"
)

type ChefMockClient struct {
	getItem   func(databagName string, databagItem string) (item chef.DataBagItem, err error)
	listItems func(name string) (data *chef.DataBagListResult, err error)
	getUser   func(name string) (user chef.User, err error)
}

func (mc *ChefMockClient) GetItem(databagName, databagItem string) (item chef.DataBagItem, err error) {
	return mc.getItem(databagName, databagItem)
}

func (mc *ChefMockClient) ListItems(name string) (data *chef.DataBagListResult, err error) {
	return mc.listItems(name)
}

func (mc *ChefMockClient) Get(name string) (user chef.User, err error) {
	if name == CORRECTUSER {
		user = chef.User{
			UserName: name,
		}
		err = nil
	} else {
		user = chef.User{}
		err = errors.New("message")
	}
	return user, err
}

func (mc *ChefMockClient) WithItem(_, _ string, _ error) {
	if mc != nil {
		mc.getItem = func(dataBagName, databagItemName string) (item chef.DataBagItem, err error) {
			ret := make(map[string]any)
			switch {
			case dataBagName == DatabagName && databagItemName == "item01":
				jsonstring := `{"id":"` + dataBagName + `-` + databagItemName + `","some_key":"fe7f29ede349519a1","some_password":"dolphin_123zc","some_username":"testuser"}`
				ret[databagItemName] = jsonstring
			case dataBagName == "databag03" && databagItemName == testitem:
				jsonMap := make(map[string]string)
				jsonMap["id"] = testitem
				jsonMap["findProperty"] = "foundProperty"
				return jsonMap, nil
			case dataBagName == DatabagName && databagItemName == testitem:
				return math.Inf(1), nil
			default:
				str := "https://chef.com/organizations/dev/data/" + dataBagName + "/" + databagItemName + ": 404"
				return nil, errors.New(str)
			}
			return ret, nil
		}
	}
}

func (mc *ChefMockClient) WithListItems(_ string, _ error) {
	if mc != nil {
		mc.listItems = func(databagName string) (data *chef.DataBagListResult, err error) {
			ret := make(chef.DataBagListResult)
			if databagName == DatabagName {
				jsonstring := fmt.Sprintf("https://chef.com/organizations/dev/data/%s/item01", databagName)
				ret["item01"] = jsonstring
			} else {
				return nil, fmt.Errorf("data bag not found: %s", databagName)
			}
			return &ret, nil
		}
	}
}

func (mc *ChefMockClient) WithUser(_ string, _ error) {
	if mc != nil {
		mc.getUser = func(name string) (user chef.User, err error) {
			return chef.User{
				UserName: name,
			}, nil
		}
	}
}
