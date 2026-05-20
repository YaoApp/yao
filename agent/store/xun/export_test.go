package xun

var (
	IsNilForTest             = isNil
	MarshalJSONFieldsForTest = marshalJSONFields
	NanoToTimeForTest        = nanoToTime
	TimeToNanoForTest        = timeToNano
	GetDriverForTest         = (*Xun).getDriver
	SandboxRawSQLForTest     = (*Xun).sandboxRawSQL
	JsonContainsValueForTest = (*Xun).jsonContainsValue
)

func NewXunForTest(store *Xun) *Xun { return store }
func EmptyXunForTest() *Xun         { return &Xun{} }
