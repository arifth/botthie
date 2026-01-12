package usecase

import (
	"fmt"
	"os"

	"github.com/arifth/botthie/config"

	"github.com/go-resty/resty/v2"
)

func PostToConfluence(data interface{}) (res resty.Response, err error) {

	// Example Request body to Confluence
	// baseURL : /content
	// Header Mandatory : Authorization : Basic base64(credential)
	//	{
	//     "type": "page",
	//     "title": "Save Data BPJSTK",
	//     "ancestors": [
	//         {
	//             "id": {{parentID}}
	//         }
	//     ],
	//     "space": {
	//         "key": "OOAPD"
	//     },
	//     "body": {
	//         "storage": {
	//             "value": "<p>This is a Check Status Download</p>",
	//             "representation": "storage"
	//         }
	//     }
	// }

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
	// if err != nil {
	// 	log.Fatal("error while posting to coinfluence \n", err)
	// 	return nil,err
	// }
	return *resConflu, nil
}
