package usecase

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/arifth/botthie/config"
	"github.com/arifth/botthie/model"
	"github.com/arifth/botthie/util"
	"github.com/go-resty/resty/v2"
)

type ListSuccess struct {
	success []resty.Response
	error   []string
}

func (Usecase) PostBulkToConfluence(collection model.PostmanCollection, templ string, uc *Usecase) (ListSuccess, error) {
	// iterate over collection item
	//TODO : flatten the array

	// post parent conflu page
	bodyReq := model.ConfluencePage{
		Type:      "page",
		Title:     "F105" + collection.Info.Name + "  " + util.GenerateRandomChars(),
		Ancestors: []model.Ancestor{{ID: os.Getenv("PARENT_ID")}},
		Space:     model.Space{Key: os.Getenv("SPACE_KEY")},
		Body: model.BodyWrapper{
			Storage: model.Storage{
				Value:          "",
				Representation: "storage",
			},
		},
	}
	type response struct {
		ID     string `json:"id"`
		Type   string `json:"type"`
		Status string `json:"status"`
		Title  string `json:"title"`
	}
	postParent, err := PostToConfluence(bodyReq, true)
	var res response
	err = json.Unmarshal(postParent.Body(), &res)
	if err != nil {
		fmt.Println(err)
	}
	parentID := res.ID
	fmt.Println(parentID)
	if err != nil {
		fmt.Println(err)
	}
	list := ListSuccess{}
	for _, item := range collection.Item {

		html := uc.ConvertToHTML(collection, templ, item)

		bodyReq := model.ConfluencePage{
			Type:      "page",
			Title:     item.Name + " " + util.GenerateRandomChars(),
			Ancestors: []model.Ancestor{{ID: parentID}},
			Space:     model.Space{Key: os.Getenv("SPACE_KEY")},
			Body: model.BodyWrapper{
				Storage: model.Storage{
					Value:          html,
					Representation: "storage",
				},
			},
		}
		//	//TODO: map value to struct
		reqBody, err := json.Marshal(bodyReq)
		if err != nil {
			fmt.Println("error when marshalling req body", err)
		}
		resConflu, err := PostToConfluence(string(reqBody), false)

		if err != nil {
			list.error = append(list.error, err.Error())
			//SendMessage(client, chatJID, fmt.Sprintf("Failed to prepare API request: %v", err))
		}
		list.success = append(list.success, resConflu)
		//link, err := getSpaceLinks(&resConflu)

	}
	return list, nil
}

func PostToConfluence(data interface{}, isParent bool) (res resty.Response, err error) {
	var baseURL = os.Getenv("BASE_URL")
	var PAT_TOKEN = os.Getenv("PAT_TOKEN")
	var hdrs = map[string]string{
		"Authorization": fmt.Sprintf("Bearer %v", PAT_TOKEN),
	}

	var conf = config.Config{
		BaseURL: baseURL,
		Headers: hdrs,
	}

	fmt.Println(data)

	var clt = config.NewClient(&conf)
	var resConflu, _ = clt.Post("/content/", data)
	return *resConflu, nil
}
