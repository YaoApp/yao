package wework

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/maps"
)

func TestWework(t *testing.T) {

	msgEncrypt := "meqbMyPr58hNy0j0YDdG9UT60UJZSh/tb3KOZt3z2SCKr6uvmSLbEnUCM89iFXS0BLWn11FOrD/xXsGUlVUSBw=="
	encodingAESKey := "RhH75tStMzrH8bMxkTw8BrBfr0ZWULL5himUaRWCs7H"

	res, err := Decrypt(encodingAESKey, msgEncrypt, false)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "8446271472585838141", res["message"])
	assert.Equal(t, "wwe146299c731e6301", res["receiveid"])
}

func TestWeworkProcess(t *testing.T) {

	msgEncrypt := "meqbMyPr58hNy0j0YDdG9UT60UJZSh/tb3KOZt3z2SCKr6uvmSLbEnUCM89iFXS0BLWn11FOrD/xXsGUlVUSBw=="
	encodingAESKey := "RhH75tStMzrH8bMxkTw8BrBfr0ZWULL5himUaRWCs7H"

	args := []interface{}{encodingAESKey, msgEncrypt}
	res := process.New("yao.wework.Decrypt", args...).Run().(map[string]interface{})

	assert.Equal(t, "8446271472585838141", res["message"])
	assert.Equal(t, "wwe146299c731e6301", res["receiveid"])
}

func TestWeworkParseXML(t *testing.T) {

	xml := `
	<xml>
		<ToUserName><![CDATA[wx5823bf96d3bd56c7]]></ToUserName>
		<FromUserName><![CDATA[mycreate]]></FromUserName>
		<CreateTime>1409659813</CreateTime>
		<MsgType><![CDATA[text]]></MsgType>
		<Content><![CDATA[hello]]></Content>
		<MsgId>4561255354251345929</MsgId>
		<AgentID>218</AgentID>
		<Nest>
			<Id>111</Id>
		</Nest>
	</xml>`

	data, err := parseXML(xml)
	if err != nil {
		t.Fatal(err)
	}

	res := maps.Of(data).Dot()
	assert.Equal(t, "218", res.Get("xml.AgentID"))
	assert.Equal(t, "111", res.Get("xml.Nest.Id"))
}
