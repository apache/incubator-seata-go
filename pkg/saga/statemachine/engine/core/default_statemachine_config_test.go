package core

import (
	"errors"
	"github.com/seata/seata-go/pkg/saga/statemachine/statelang"
	"github.com/stretchr/testify/assert"
	"io"
	"path/filepath"
	"testing"
)

func TestDefaultStateMachineConfig_LoadValidJSON(t *testing.T) {
	config := NewDefaultStateMachineConfig()
	testFile := filepath.Join("testdata", "order_saga.json")

	err := config.LoadConfig(testFile)
	assert.NoError(t, err, "加载 JSON 配置应成功")

	smo := config.GetStateMachineDefinition("OrderSaga")
	assert.NotNil(t, smo, "状态机定义不应为空")
	assert.Equal(t, "CreateOrder", smo.StartState, "起始状态应正确")
	assert.Contains(t, smo.States, "CreateOrder", "状态节点应存在")

	assert.Equal(t, 30000, config.TransOperationTimeout, "超时时间应正确读取")
}

func TestDefaultStateMachineConfig_LoadValidYAML(t *testing.T) {
	config := NewDefaultStateMachineConfig()
	testFile := filepath.Join("testdata", "order_saga.yaml")

	err := config.LoadConfig(testFile)
	assert.NoError(t, err, "加载 YAML 配置应成功")

	smo := config.GetStateMachineDefinition("OrderSaga")
	assert.NotNil(t, smo)
}

func TestLoadNonExistentFile(t *testing.T) {
	config := NewDefaultStateMachineConfig()
	err := config.LoadConfig("non_existent.json")
	assert.Error(t, err, "加载不存在的文件应报错")
	assert.Contains(t, err.Error(), "failed to read config file", "错误信息应包含文件读取失败")
}

func TestGetStateMachineDefinition_Exists(t *testing.T) {
	config := NewDefaultStateMachineConfig()
	_ = config.LoadConfig(filepath.Join("testdata", "order_saga.json"))

	smo := config.GetStateMachineDefinition("OrderSaga")
	assert.NotNil(t, smo)
	assert.Equal(t, "1.0", smo.Version, "版本号应正确")
}

func TestGetNonExistentStateMachine(t *testing.T) {
	config := NewDefaultStateMachineConfig()
	smo := config.GetStateMachineDefinition("NonExistent")
	assert.Nil(t, smo, "未加载的状态机应返回 nil")
}

func TestLoadDuplicateStateMachine(t *testing.T) {
	config := NewDefaultStateMachineConfig()
	testFile := filepath.Join("testdata", "order_saga.json")

	err := config.LoadConfig(testFile)
	assert.NoError(t, err)

	err = config.LoadConfig(testFile)
	assert.Error(t, err, "重复加载应触发名称冲突")
	assert.Contains(t, err.Error(), "already exists", "错误信息应包含冲突提示")
}

func TestRuntimeConfig_OverrideDefaults(t *testing.T) {
	config := NewDefaultStateMachineConfig()
	assert.Equal(t, "UTF-8", config.Charset, "默认字符集应为 UTF-8")

	_ = config.LoadConfig(filepath.Join("testdata", "order_saga.json"))
	assert.Equal(t, "UTF-8", config.Charset, "若配置未指定，应使用默认值")

	customConfig := &RuntimeConfig{
		Charset: "GBK",
	}
	config.applyRuntimeConfig(customConfig)
	assert.Equal(t, "GBK", config.Charset, "运行时参数应正确覆盖")
}

func TestGetDefaultExpressionFactory(t *testing.T) {
	config := NewDefaultStateMachineConfig()
	factory := config.GetExpressionFactory("el")
	assert.NotNil(t, factory, "默认 EL 工厂应存在")

	unknownFactory := config.GetExpressionFactory("unknown")
	assert.Nil(t, unknownFactory, "未知表达式类型应返回 nil")
}

func TestGetServiceInvoker(t *testing.T) {
	config := NewDefaultStateMachineConfig()
	invoker := config.GetServiceInvoker("local")
	assert.NotNil(t, invoker, "默认本地调用器应存在")
}

func TestLoadConfig_InvalidJSON(t *testing.T) {
	config := NewDefaultStateMachineConfig()
	testFile := filepath.Join("testdata", "invalid.json")

	err := config.LoadConfig(testFile)
	assert.Error(t, err, "加载无效的 JSON 配置应报错")
	assert.Contains(t, err.Error(), "failed to parse state machine definition", "错误信息应包含解析失败")
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	config := NewDefaultStateMachineConfig()
	testFile := filepath.Join("testdata", "invalid.yaml")

	err := config.LoadConfig(testFile)
	assert.Error(t, err, "加载无效的 YAML 配置应报错")
	assert.Contains(t, err.Error(), "failed to parse state machine definition", "错误信息应包含解析失败")
}

func TestRegisterStateMachineDef_Fail(t *testing.T) {
	config := NewDefaultStateMachineConfig()
	invalidResource := []string{"invalid_path.json"}

	err := config.RegisterStateMachineDef(invalidResource)
	assert.Error(t, err, "注册无效资源应报错")
	assert.Contains(t, err.Error(), "open resource file failed", "错误信息应包含文件打开失败")
}

func TestInit_ExpressionFactoryManagerNil(t *testing.T) {
	config := NewDefaultStateMachineConfig()
	config.expressionFactoryManager = nil

	err := config.Init()
	assert.NoError(t, err, "表达式工厂管理器为 nil 时初始化应成功")
}

func TestInit_ServiceInvokerManagerNil(t *testing.T) {
	config := NewDefaultStateMachineConfig()
	config.serviceInvokerManager = nil

	err := config.Init()
	assert.NoError(t, err, "服务调用器管理器为 nil 时初始化应成功")
}

func TestInit_StateMachineRepositoryNil(t *testing.T) {
	config := NewDefaultStateMachineConfig()
	config.stateMachineRepository = nil

	err := config.Init()
	assert.NoError(t, err, "状态机仓库为 nil 时初始化应成功")
}

func TestApplyRuntimeConfig_BoundaryValues(t *testing.T) {
	config := NewDefaultStateMachineConfig()
	customConfig := &RuntimeConfig{
		TransOperationTimeout: 1,
		ServiceInvokeTimeout:  1,
	}
	config.applyRuntimeConfig(customConfig)
	assert.Equal(t, 1, config.TransOperationTimeout, "最小事务操作超时时间应正确应用")
	assert.Equal(t, 1, config.ServiceInvokeTimeout, "最小服务调用超时时间应正确应用")

	maxTimeout := int(^uint(0) >> 1)
	customConfig = &RuntimeConfig{
		TransOperationTimeout: maxTimeout,
		ServiceInvokeTimeout:  maxTimeout,
	}
	config.applyRuntimeConfig(customConfig)
	assert.Equal(t, maxTimeout, config.TransOperationTimeout, "最大事务操作超时时间应正确应用")
	assert.Equal(t, maxTimeout, config.ServiceInvokeTimeout, "最大服务调用超时时间应正确应用")
}

type TestStateMachineRepositoryMock struct{}

func (m *TestStateMachineRepositoryMock) GetStateMachineById(stateMachineId string) (statelang.StateMachine, error) {
	return nil, errors.New("get state machine by id failed")
}

func (m *TestStateMachineRepositoryMock) GetStateMachineByNameAndTenantId(stateMachineName string, tenantId string) (statelang.StateMachine, error) {
	return nil, errors.New("get state machine by name and tenant id failed")
}

func (m *TestStateMachineRepositoryMock) GetLastVersionStateMachine(stateMachineName string, tenantId string) (statelang.StateMachine, error) {
	return nil, errors.New("get last version state machine failed")
}

func (m *TestStateMachineRepositoryMock) RegistryStateMachine(machine statelang.StateMachine) error {
	return errors.New("registry state machine failed")
}

func (m *TestStateMachineRepositoryMock) RegistryStateMachineByReader(reader io.Reader) error {
	return errors.New("registry state machine by reader failed")
}

func TestRegisterStateMachineDef_RepositoryError(t *testing.T) {
	config := NewDefaultStateMachineConfig()
	config.stateMachineRepository = &TestStateMachineRepositoryMock{}
	resource := []string{"testdata/test.json"}

	err := config.RegisterStateMachineDef(resource)
	assert.Error(t, err, "状态机仓库报错时注册应失败")
	assert.Contains(t, err.Error(), "register state machine from file failed", "错误信息应包含注册失败")
}
