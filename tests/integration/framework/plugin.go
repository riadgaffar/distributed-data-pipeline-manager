package framework

type PluginTest interface {
	Setup() error
	Execute() error
	Validate() error
	Teardown() error
}

type BasePluginTest struct {
	Framework *TestFramework
}

func NewBasePluginTest(framework *TestFramework) *BasePluginTest {
	return &BasePluginTest{
		Framework: framework,
	}
}
