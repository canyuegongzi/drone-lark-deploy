package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type Element struct {
	Tag      string            `json:"tag,omitempty"`
	Content  string            `json:"content,omitempty"`
	Elements []Element         `json:"elements,omitempty"`
	I18N     map[string]string `json:"i18n,omitempty"`
}

type Header struct {
	Template string  `json:"template,omitempty"`
	Title    Element `json:"title,omitempty"`
}

type Card struct {
	I18nElements map[string][]Element `json:"i18n_elements,omitempty"`
	Header       Header               `json:"header,omitempty"`
}

type Body struct {
	Timestamp int64  `json:"timestamp"`
	Sign      string `json:"sign"`
	MsgType   string `json:"msg_type"`
	Card      Card   `json:"card"`
}

func main() {
	webhook := os.Getenv("PLUGIN_WEBHOOK")
	if webhook == "" {
		fmt.Println("Missing webhook configuration")
		return
	}

	// 通知的类型
	noticetype := os.Getenv("PLUGIN_MESSAGETYPE")
	dockergroup := os.Getenv("PLUGIN_DOCKERGROUP")

	if noticetype == "" {
		noticetype = "BUILD"
	}

	if dockergroup == "" {
		fmt.Println("Missing dockergroup configuration")
		return
	}

	secret := os.Getenv("PLUGIN_SECRET")
	if secret == "" {
		fmt.Println("Missing secret configuration")
		return
	}

	timestamp := time.Now().Unix()
	sign := generateSignature(timestamp, secret)

	repo := os.Getenv("DRONE_REPO_NAME")

	var color string
	var cnTitle string
	var enTitle string

	buildNo := os.Getenv("DRONE_BUILD_NUMBER")

	if noticetype == "BUILD" {
		if os.Getenv("DRONE_BUILD_STATUS") == "success" {
			color = "green"
			cnTitle = "✅ " + repo + " 构建成功 #" + buildNo
			enTitle = "✅ " + repo + " Build Successfully #" + buildNo
		} else {
			color = "red"
			cnTitle = "❌ " + repo + " 构建失败 #" + buildNo
			enTitle = "❌ " + repo + " Build Failed #" + buildNo
		}

		header := Header{
			Template: color,
			Title: Element{
				Tag: "plain_text",
				I18N: map[string]string{
					"zh_cn": cnTitle,
					"en_us": enTitle,
				},
			},
		}

		var cnMarkdown strings.Builder
		var enMarkdown strings.Builder

		if os.Getenv("DRONE_FAILED_STEPS") != "" {
			cnMarkdown.WriteString("**:SLAP: 失败：** <font color='red'>")
			cnMarkdown.WriteString(os.Getenv("DRONE_FAILED_STEPS"))
			cnMarkdown.WriteString("</font>\n")
			enMarkdown.WriteString("**:SLAP: FAIL: ** <font color='red'>")
			enMarkdown.WriteString(os.Getenv("DRONE_FAILED_STEPS"))
			enMarkdown.WriteString("</font>\n")
		}

		cnMarkdown.WriteString("**:GeneralBusinessTrip: 项目：** [")
		cnMarkdown.WriteString(repo)
		cnMarkdown.WriteString("](")
		cnMarkdown.WriteString(os.Getenv("DRONE_REPO_LINK"))
		cnMarkdown.WriteString(")\n")

		enMarkdown.WriteString("**:GeneralBusinessTrip: PROJ: ** [")
		enMarkdown.WriteString(repo)
		enMarkdown.WriteString("](")
		enMarkdown.WriteString(os.Getenv("DRONE_REPO_LINK"))
		enMarkdown.WriteString(")\n")

		if os.Getenv("DRONE_TAG") != "" {
			cnMarkdown.WriteString("**:Pin: 标签：** <text_tag color='indigo'>")
			cnMarkdown.WriteString(os.Getenv("DRONE_TAG"))
			cnMarkdown.WriteString("</text_tag>\n")

			enMarkdown.WriteString("**:Pin: TAGS: ** <text_tag color='indigo'>")
			enMarkdown.WriteString(os.Getenv("DRONE_TAG"))
			enMarkdown.WriteString("</text_tag>\n")

		} else if os.Getenv("DRONE_REPO_BRANCH") != "" {

			cnMarkdown.WriteString("**:StatusReading: 分支：** <text_tag color='blue'>")
			cnMarkdown.WriteString(os.Getenv("DRONE_REPO_BRANCH"))
			cnMarkdown.WriteString("</text_tag>\n")

			enMarkdown.WriteString("**:StatusReading: BCHS: ** <text_tag color='blue'>")
			enMarkdown.WriteString(os.Getenv("DRONE_REPO_BRANCH"))
			enMarkdown.WriteString("</text_tag>\n")
		}

		author := os.Getenv("DRONE_COMMIT_AUTHOR")
		authorName := os.Getenv("DRONE_COMMIT_AUTHOR_NAME")

		if author == "" {
			if authorName != "" {
				author = authorName
			}
		} else if authorName != "" && author != authorName {
			author = authorName + "@" + author
		}

		if author != "" {
			email := os.Getenv("DRONE_COMMIT_AUTHOR_EMAIL")
			hasEmail := email != ""
			cnMarkdown.WriteString("**:EMBARRASSED: 提交：** ")
			enMarkdown.WriteString("**:EMBARRASSED: CMMT: ** ")
			if hasEmail {
				cnMarkdown.WriteString("[")
				enMarkdown.WriteString("[")
			}
			cnMarkdown.WriteString(author)
			enMarkdown.WriteString(author)
			if hasEmail {
				cnMarkdown.WriteString("](mailto:")
				cnMarkdown.WriteString(email)
				cnMarkdown.WriteString(")")

				enMarkdown.WriteString("](mailto:")
				enMarkdown.WriteString(email)
				enMarkdown.WriteString(")")
			}
			cnMarkdown.WriteString("\n")
			enMarkdown.WriteString("\n")
		}

		if os.Getenv("DRONE_COMMIT_SHA") != "" {
			cnMarkdown.WriteString("**:Status_PrivateMessage: 信息：** [#")
			cnMarkdown.WriteString(os.Getenv("DRONE_COMMIT_SHA")[:8])
			cnMarkdown.WriteString("](")
			cnMarkdown.WriteString(os.Getenv("DRONE_COMMIT_LINK"))
			cnMarkdown.WriteString(")\n")

			enMarkdown.WriteString("**:Status_PrivateMessage: NOTE: ** [#")
			enMarkdown.WriteString(os.Getenv("DRONE_COMMIT_SHA")[:8])
			enMarkdown.WriteString("](")
			enMarkdown.WriteString(os.Getenv("DRONE_COMMIT_LINK"))
			enMarkdown.WriteString(")\n")
		}

		cnMarkdown.WriteString(" ---\n")
		cnMarkdown.WriteString(os.Getenv("DRONE_COMMIT_MESSAGE"))

		enMarkdown.WriteString(" ---\n")
		enMarkdown.WriteString(os.Getenv("DRONE_COMMIT_MESSAGE"))

		cnElements := []Element{
			{
				Tag:     "markdown",
				Content: cnMarkdown.String(),
			},
			{
				Tag: "note",
				Elements: []Element{
					{
						Tag:     "lark_md",
						Content: ":Loudspeaker: [以上信息由 drone 飞书机器人自动发出](" + os.Getenv("DRONE_BUILD_LINK") + ")",
					},
				},
			},
		}

		enElements := []Element{
			{
				Tag:     "markdown",
				Content: enMarkdown.String(),
			},
			{
				Tag: "note",
				Elements: []Element{
					{
						Tag:     "lark_md",
						Content: ":Loudspeaker: [This msg is sent by drone lark robot](" + os.Getenv("DRONE_BUILD_LINK") + ")",
					},
				},
			},
		}

		body := Body{
			Timestamp: timestamp,
			Sign:      sign,
			MsgType:   "interactive",
			Card: Card{
				Header: header,
				I18nElements: map[string][]Element{
					"zh_cn": cnElements,
					"en_us": enElements,
				},
			},
		}

		err := sendRequest(webhook, body)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
	}

	if noticetype == "DEPLOY" {
		if os.Getenv("DRONE_BUILD_STATUS") == "success" {
			color = "green"
			cnTitle = "✅ " + repo + " 部署成功 #" + buildNo
			enTitle = "✅ " + repo + " Deploy Successfully #" + buildNo
		} else {
			color = "red"
			cnTitle = "❌ " + repo + " 部署失败 #" + buildNo
			enTitle = "❌ " + repo + " Deploy Failed #" + buildNo
		}

		header := Header{
			Template: color,
			Title: Element{
				Tag: "plain_text",
				I18N: map[string]string{
					"zh_cn": cnTitle,
					"en_us": enTitle,
				},
			},
		}

		var cnMarkdown strings.Builder
		var enMarkdown strings.Builder

		if os.Getenv("DRONE_FAILED_STEPS") != "" {
			cnMarkdown.WriteString("**:SLAP: 失败：** <font color='red'>")
			cnMarkdown.WriteString(os.Getenv("DRONE_FAILED_STEPS"))
			cnMarkdown.WriteString("</font>\n")
			enMarkdown.WriteString("**:SLAP: FAIL: ** <font color='red'>")
			enMarkdown.WriteString(os.Getenv("DRONE_FAILED_STEPS"))
			enMarkdown.WriteString("</font>\n")
		}

		cnMarkdown.WriteString("**:GeneralBusinessTrip: 项目：** [")
		cnMarkdown.WriteString(repo)
		cnMarkdown.WriteString("](")
		cnMarkdown.WriteString(os.Getenv("DRONE_REPO_LINK"))
		cnMarkdown.WriteString(")\n")

		enMarkdown.WriteString("**:GeneralBusinessTrip: PROJ: ** [")
		enMarkdown.WriteString(repo)
		enMarkdown.WriteString("](")
		enMarkdown.WriteString(os.Getenv("DRONE_REPO_LINK"))
		enMarkdown.WriteString(")\n")

		if os.Getenv("DRONE_TAG") != "" {
			cnMarkdown.WriteString("**:Pin: 标签：** <text_tag color='indigo'>")
			cnMarkdown.WriteString(os.Getenv("DRONE_TAG"))
			cnMarkdown.WriteString("</text_tag>\n")

			enMarkdown.WriteString("**:Pin: TAGS: ** <text_tag color='indigo'>")
			enMarkdown.WriteString(os.Getenv("DRONE_TAG"))
			enMarkdown.WriteString("</text_tag>\n")

		} else if os.Getenv("DRONE_REPO_BRANCH") != "" {

			cnMarkdown.WriteString("**:StatusReading: 分支：** <text_tag color='blue'>")
			cnMarkdown.WriteString(os.Getenv("DRONE_REPO_BRANCH"))
			cnMarkdown.WriteString("</text_tag>\n")

			enMarkdown.WriteString("**:StatusReading: BCHS: ** <text_tag color='blue'>")
			enMarkdown.WriteString(os.Getenv("DRONE_REPO_BRANCH"))
			enMarkdown.WriteString("</text_tag>\n")
		}

		author := os.Getenv("DRONE_COMMIT_AUTHOR")
		authorName := os.Getenv("DRONE_COMMIT_AUTHOR_NAME")

		if author == "" {
			if authorName != "" {
				author = authorName
			}
		} else if authorName != "" && author != authorName {
			author = authorName + "@" + author
		}

		if author != "" {
			email := os.Getenv("DRONE_COMMIT_AUTHOR_EMAIL")
			hasEmail := email != ""
			cnMarkdown.WriteString("**:EMBARRASSED: 提交：** ")
			enMarkdown.WriteString("**:EMBARRASSED: CMMT: ** ")
			if hasEmail {
				cnMarkdown.WriteString("[")
				enMarkdown.WriteString("[")
			}
			cnMarkdown.WriteString(author)
			enMarkdown.WriteString(author)
			if hasEmail {
				cnMarkdown.WriteString("](mailto:")
				cnMarkdown.WriteString(email)
				cnMarkdown.WriteString(")")

				enMarkdown.WriteString("](mailto:")
				enMarkdown.WriteString(email)
				enMarkdown.WriteString(")")
			}
			cnMarkdown.WriteString("\n")
			enMarkdown.WriteString("\n")
		}

		if os.Getenv("DRONE_COMMIT_SHA") != "" {
			cnMarkdown.WriteString("**:Status_PrivateMessage: 信息：** [#")
			cnMarkdown.WriteString(os.Getenv("DRONE_COMMIT_SHA")[:8])
			cnMarkdown.WriteString("](")
			cnMarkdown.WriteString(os.Getenv("DRONE_COMMIT_LINK"))
			cnMarkdown.WriteString(")\n")

			enMarkdown.WriteString("**:Status_PrivateMessage: NOTE: ** [#")
			enMarkdown.WriteString(os.Getenv("DRONE_COMMIT_SHA")[:8])
			enMarkdown.WriteString("](")
			enMarkdown.WriteString(os.Getenv("DRONE_COMMIT_LINK"))
			enMarkdown.WriteString(")\n")
		}
		// 镜像标签
		if os.Getenv("DRONE_COMMIT_SHA") != "" {
			cnMarkdown.WriteString("**:CheckMark: 镜像：** ** ")
			cnMarkdown.WriteString(dockergroup)
			cnMarkdown.WriteString("/")
			cnMarkdown.WriteString(repo)
			cnMarkdown.WriteString(":")
			cnMarkdown.WriteString(os.Getenv("DRONE_COMMIT_SHA")[:7])
			cnMarkdown.WriteString("**\n")

			enMarkdown.WriteString("**:CheckMark: IMAGE：** ** ")
			enMarkdown.WriteString(dockergroup)
			enMarkdown.WriteString("/")
			enMarkdown.WriteString(repo)
			enMarkdown.WriteString(":")
			enMarkdown.WriteString(os.Getenv("DRONE_COMMIT_SHA")[:7])
			enMarkdown.WriteString("**\n")

		}

		cnMarkdown.WriteString(" ---\n")
		cnMarkdown.WriteString(os.Getenv("DRONE_COMMIT_MESSAGE"))

		enMarkdown.WriteString(" ---\n")
		enMarkdown.WriteString(os.Getenv("DRONE_COMMIT_MESSAGE"))

		cnElements := []Element{
			{
				Tag:     "markdown",
				Content: cnMarkdown.String(),
			},
			{
				Tag: "note",
				Elements: []Element{
					{
						Tag:     "lark_md",
						Content: ":Loudspeaker: [以上信息由 drone 飞书机器人自动发出](" + os.Getenv("DRONE_BUILD_LINK") + ")",
					},
				},
			},
		}

		enElements := []Element{
			{
				Tag:     "markdown",
				Content: enMarkdown.String(),
			},
			{
				Tag: "note",
				Elements: []Element{
					{
						Tag:     "lark_md",
						Content: ":Loudspeaker: [This msg is sent by drone lark robot](" + os.Getenv("DRONE_BUILD_LINK") + ")",
					},
				},
			},
		}

		body := Body{
			Timestamp: timestamp,
			Sign:      sign,
			MsgType:   "interactive",
			Card: Card{
				Header: header,
				I18nElements: map[string][]Element{
					"zh_cn": cnElements,
					"en_us": enElements,
				},
			},
		}

		err := sendRequest(webhook, body)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
	}

}

func generateSignature(timestamp int64, secret string) string {
	message := fmt.Sprintf("%v\n%v", timestamp, secret)
	mac := hmac.New(sha256.New, []byte(message))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	return signature
}

func sendRequest(url string, body Body) error {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return err
	}
	debug := os.Getenv("PLUGIN_DEBUG") == "true"
	if debug {
		fmt.Println("Request Body:", string(jsonBody))
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	fmt.Println("Response Status:", resp.Status)

	// 读取响应主体内容
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if debug {
		fmt.Println("Response Body:", string(responseBody))
	}
	return nil
}
