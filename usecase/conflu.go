package usecase

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/arifth/botthie/config"
	"github.com/arifth/botthie/model"

	"github.com/go-resty/resty/v2"
)

type ListSuccess struct {
	success []resty.Response
	error   []string
}

func (Usecase) PostBulkToConfluence(collection model.PostmanCollection, templ string, uc *Usecase) (ListSuccess, error) {
	// iterate over collection item
	//TODO : flatten the array
	list := ListSuccess{}

	for _, item := range collection.Item {

		html := uc.ConvertToHTML(collection, templ, item)

		bodyReq := model.ConfluencePage{
			Type:      "page",
			Title:     item.Name,
			Ancestors: []model.Ancestor{{ID: os.Getenv("PARENT_ID")}},
			Space:     model.Space{Key: os.Getenv("SPACE_KEY")},
			Body: model.BodyWrapper{
				Storage: model.Storage{
					Value:          string(html),
					Representation: "storage",
				},
			},
		}

		//TODO: map value to struct
		reqBody, err := json.Marshal(bodyReq)
		if err != nil {
			fmt.Println("error when marshalling req body", err)
		}
		resConflu, err := PostToConfluence(string(reqBody))

		if err != nil {
			list.error = append(list.error, err.Error())
			//SendMessage(client, chatJID, fmt.Sprintf("Failed to prepare API request: %v", err))
		}
		list.success = append(list.success, resConflu)
		//link, err := getSpaceLinks(&resConflu)

	}
	return list, nil
}

func PostToConfluence(data interface{}) (res resty.Response, err error) {
	var baseURL = os.Getenv("BASE_URL")
	var BASE64 = os.Getenv("BASIC_AUTH")
	// var hdrs = map[string][string]{"Authorization": fmt.Sprintf("Basic %v",BASE64)}
	var hdrs = map[string]string{
		// Now you don't need the curly braces around the value
		"Authorization": fmt.Sprintf("Basic %v", BASE64),
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
