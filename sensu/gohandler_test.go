package sensu

import (
	"fmt"
	"github.com/sensu/sensu-go/types"
	"github.com/stretchr/testify/assert"
	"io"
	"os"
	"testing"
)

type handlerValues struct {
	arg1 string
	arg2 uint64
	arg3 bool
}

var (
	defaultHandlerConfig = HandlerConfig{
		Name:     "TestHandler",
		Short:    "Short Description",
		Timeout:  10,
		Keyspace: "sensu.io/plugins/segp/config",
	}

	defaultOption1 = HandlerConfigOption{
		Argument:  "arg1",
		Default:   "Default1",
		Env:       "ENV_1",
		Path:      "path1",
		Shorthand: "d",
		Usage:     "First argument",
	}

	defaultOption2 = HandlerConfigOption{
		Argument:  "arg2",
		Default:   uint64(33333),
		Env:       "ENV_2",
		Path:      "path2",
		Shorthand: "e",
		Usage:     "Second argument",
	}

	defaultOption3 = HandlerConfigOption{
		Argument:  "arg3",
		Default:   false,
		Env:       "ENV_3",
		Path:      "path3",
		Shorthand: "f",
		Usage:     "Third argument",
	}

	defaultCmdLineArgs = []string{"--arg1", "value-arg1", "--arg2", "7531", "--arg3=false"}
)

func TestNewGoHandler(t *testing.T) {
	options := getDefaultOptions()
	goHandler := NewGoHandler(&defaultHandlerConfig, options, func(event *types.Event) error {
		return nil
	}, func(event *types.Event) error {
		return nil
	})

	assert.NotNil(t, goHandler)
	assert.NotNil(t, goHandler.options)
	assert.Equal(t, options, goHandler.options)
	assert.NotNil(t, goHandler.config)
	assert.Equal(t, &defaultHandlerConfig, goHandler.config)
	assert.NotNil(t, goHandler.validationFunction)
	assert.NotNil(t, goHandler.executeFunction)
	assert.Nil(t, goHandler.sensuEvent)
	assert.Equal(t, os.Stdin, goHandler.eventReader)
	assert.NotNil(t, goHandler.cmdArgs)
}

func TestSetOptionValue_String(t *testing.T) {
	finalValue := ""
	option := defaultOption1
	option.Value = &finalValue
	err := setOptionValue(&option, "abc")
	assert.Nil(t, err)
	assert.Equal(t, "abc", finalValue)
}

func TestSetOptionValue_EmptyString(t *testing.T) {
	finalValue := ""
	option := defaultOption1
	option.Value = &finalValue
	err := setOptionValue(&option, "")
	assert.Nil(t, err)
	assert.Equal(t, "", finalValue)
}

func TestSetOptionValue_ValidUint64(t *testing.T) {
	var finalValue uint64
	option := defaultOption1
	option.Value = &finalValue
	err := setOptionValue(&option, "123")
	assert.Nil(t, err)
	assert.Equal(t, uint64(123), finalValue)
}

func TestSetOptionValue_InvalidUint64(t *testing.T) {
	var finalValue uint64
	option := defaultOption1
	option.Value = &finalValue
	err := setOptionValue(&option, "abc")
	assert.NotNil(t, err)
	assert.Equal(t, uint64(0), finalValue)
}

func TestSetOptionValue_TrueBool(t *testing.T) {
	var finalValue bool
	option := defaultOption1
	option.Value = &finalValue
	err := setOptionValue(&option, "true")
	assert.Nil(t, err)
	assert.Equal(t, true, finalValue)
}

func TestSetOptionValue_FalseBool(t *testing.T) {
	finalValue := true
	option := defaultOption1
	option.Value = &finalValue
	err := setOptionValue(&option, "false")
	assert.Nil(t, err)
	assert.Equal(t, false, finalValue)
}

func TestSetOptionValue_InvalidBool(t *testing.T) {
	var finalValue bool
	option := defaultOption1
	option.Value = &finalValue
	err := setOptionValue(&option, "yes")
	assert.NotNil(t, err)
	assert.Equal(t, false, finalValue)
}

func goHandlerExecuteUtil(t *testing.T, handlerConfig *HandlerConfig, eventFile string, cmdLineArgs []string,
	validationFunction func(*types.Event) error, executeFunction func(*types.Event) error,
	expectedValue1 interface{}, expectedValue2 interface{}, expectedValue3 interface{}) error {
	options := getDefaultOptions()
	values := handlerValues{}
	options[0].Value = &values.arg1
	options[1].Value = &values.arg2
	options[2].Value = &values.arg3

	goHandler := NewGoHandler(handlerConfig, options, validationFunction, executeFunction)

	if len(cmdLineArgs) > 0 {
		goHandler.cmdArgs.SetArgs(cmdLineArgs)
	} else {
		goHandler.cmdArgs.SetArgs([]string{})
	}

	// Replace stdin reader with file reader
	goHandler.eventReader = getFileReader(eventFile)
	err := goHandler.Execute()

	assert.Equal(t, expectedValue1, values.arg1)
	assert.Equal(t, expectedValue2, values.arg2)
	assert.Equal(t, expectedValue3, values.arg3)

	return err
}

// Test check override
func TestGoHandler_Execute_Check(t *testing.T) {
	var validateCalled, executeCalled bool
	clearEnvironment()
	err := goHandlerExecuteUtil(t, &defaultHandlerConfig, "test/event-check-override.json", nil,
		func(event *types.Event) error {
			validateCalled = true
			assert.NotNil(t, event)
			return nil
		}, func(event *types.Event) error {
			executeCalled = true
			assert.NotNil(t, event)
			return nil
		},
		"value-check1", uint64(1357), false)
	assert.Nil(t, err)
	assert.True(t, validateCalled)
	assert.True(t, executeCalled)
}

// Test check override with invalid value
func TestGoHandler_Execute_CheckInvalidValue(t *testing.T) {
	var validateCalled, executeCalled bool
	clearEnvironment()
	err := goHandlerExecuteUtil(t, &defaultHandlerConfig, "test/event-check-override-invalid-value.json", nil,
		func(event *types.Event) error {
			validateCalled = true
			assert.NotNil(t, event)
			return nil
		}, func(event *types.Event) error {
			executeCalled = true
			assert.NotNil(t, event)
			return nil
		},
		"value-check1", uint64(33333), false)
	assert.NotNil(t, err)
	assert.False(t, validateCalled)
	assert.False(t, executeCalled)
}

// Test entity override
func TestGoHandler_Execute_Entity(t *testing.T) {
	var validateCalled, executeCalled bool
	clearEnvironment()
	err := goHandlerExecuteUtil(t, &defaultHandlerConfig, "test/event-entity-override.json", nil,
		func(event *types.Event) error {
			validateCalled = true
			assert.NotNil(t, event)
			return nil
		}, func(event *types.Event) error {
			executeCalled = true
			assert.NotNil(t, event)
			return nil
		},
		"value-entity1", uint64(2468), true)

	assert.Nil(t, err)
	assert.True(t, validateCalled)
	assert.True(t, executeCalled)
}

// Test entity override - invalid value
func TestGoHandler_Execute_EntityInvalidValue(t *testing.T) {
	var validateCalled, executeCalled bool
	clearEnvironment()
	err := goHandlerExecuteUtil(t, &defaultHandlerConfig, "test/event-entity-override-invalid-value.json", nil,
		func(event *types.Event) error {
			validateCalled = true
			assert.NotNil(t, event)
			return nil
		}, func(event *types.Event) error {
			executeCalled = true
			assert.NotNil(t, event)
			return nil
		},
		"value-entity1", uint64(33333), false)

	assert.NotNil(t, err)
	assert.False(t, validateCalled)
	assert.False(t, executeCalled)
}

// Test environment
func TestGoHandler_Execute_Environment(t *testing.T) {
	var validateCalled, executeCalled bool
	clearEnvironment()
	_ = os.Setenv("ENV_1", "value-env1")
	_ = os.Setenv("ENV_2", "9753")
	_ = os.Setenv("ENV_3", "true")
	err := goHandlerExecuteUtil(t, &defaultHandlerConfig, "test/event-no-override.json", nil,
		func(event *types.Event) error {
			validateCalled = true
			assert.NotNil(t, event)
			return nil
		}, func(event *types.Event) error {
			executeCalled = true
			assert.NotNil(t, event)
			return nil
		},
		"value-env1", uint64(9753), true)
	assert.Nil(t, err)
	assert.True(t, validateCalled)
	assert.True(t, executeCalled)
}

// Test cmd line arguments
func TestGoHandler_Execute_CmdLineArgs(t *testing.T) {
	var validateCalled, executeCalled bool
	clearEnvironment()
	err := goHandlerExecuteUtil(t, &defaultHandlerConfig, "test/event-no-override.json", defaultCmdLineArgs,
		func(event *types.Event) error {
			validateCalled = true
			assert.NotNil(t, event)
			return nil
		}, func(event *types.Event) error {
			executeCalled = true
			assert.NotNil(t, event)
			return nil
		},
		"value-arg1", uint64(7531), false)
	assert.Nil(t, err)
	assert.True(t, validateCalled)
	assert.True(t, executeCalled)
}

// Test check priority - check override
func TestGoHandler_Execute_PriorityCheck(t *testing.T) {
	var validateCalled, executeCalled bool
	clearEnvironment()
	_ = os.Setenv("ENV_1", "value-env1")
	_ = os.Setenv("ENV_2", "9753")
	_ = os.Setenv("ENV_3", "true")
	err := goHandlerExecuteUtil(t, &defaultHandlerConfig, "test/event-check-entity-override.json", defaultCmdLineArgs,
		func(event *types.Event) error {
			validateCalled = true
			assert.NotNil(t, event)
			return nil
		}, func(event *types.Event) error {
			executeCalled = true
			assert.NotNil(t, event)
			return nil
		},
		"value-check1", uint64(1357), false)
	assert.Nil(t, err)
	assert.True(t, validateCalled)
	assert.True(t, executeCalled)
}

// Test next priority - entity override
func TestGoHandler_Execute_PriorityEntity(t *testing.T) {
	var validateCalled, executeCalled bool
	clearEnvironment()
	_ = os.Setenv("ENV_1", "value-env1")
	_ = os.Setenv("ENV_2", "9753")
	_ = os.Setenv("ENV_3", "true")
	err := goHandlerExecuteUtil(t, &defaultHandlerConfig, "test/event-entity-override.json", defaultCmdLineArgs,
		func(event *types.Event) error {
			validateCalled = true
			assert.NotNil(t, event)
			return nil
		}, func(event *types.Event) error {
			executeCalled = true
			assert.NotNil(t, event)
			return nil
		},
		"value-entity1", uint64(2468), true)
	assert.Nil(t, err)
	assert.True(t, validateCalled)
	assert.True(t, executeCalled)
}

// Test next priority - cmd line arguments
func TestGoHandler_Execute_PriorityCmdLine(t *testing.T) {
	var validateCalled, executeCalled bool
	clearEnvironment()
	_ = os.Setenv("ENV_1", "value-env1")
	_ = os.Setenv("ENV_2", "9753")
	_ = os.Setenv("ENV_3", "true")
	err := goHandlerExecuteUtil(t, &defaultHandlerConfig, "test/event-no-override.json", defaultCmdLineArgs,
		func(event *types.Event) error {
			validateCalled = true
			assert.NotNil(t, event)
			return nil
		}, func(event *types.Event) error {
			executeCalled = true
			assert.NotNil(t, event)
			return nil
		},
		"value-arg1", uint64(7531), false)
	assert.Nil(t, err)
	assert.True(t, validateCalled)
	assert.True(t, executeCalled)
}

// Test validation error
func TestGoHandler_Execute_ValidationError(t *testing.T) {
	var validateCalled, executeCalled bool
	clearEnvironment()
	err := goHandlerExecuteUtil(t, &defaultHandlerConfig, "test/event-no-override.json", defaultCmdLineArgs,
		func(event *types.Event) error {
			validateCalled = true
			assert.NotNil(t, event)
			return fmt.Errorf("validation error")
		}, func(event *types.Event) error {
			executeCalled = true
			return nil
		},
		"value-arg1", uint64(7531), false)
	assert.NotNil(t, err)
	assert.EqualError(t, err, "error validating input: validation error")
	assert.True(t, validateCalled)
	assert.False(t, executeCalled)
}

// Test execute error
func TestGoHandler_Execute_ExecuteError(t *testing.T) {
	var validateCalled, executeCalled bool
	clearEnvironment()
	err := goHandlerExecuteUtil(t, &defaultHandlerConfig, "test/event-no-override.json", defaultCmdLineArgs,
		func(event *types.Event) error {
			validateCalled = true
			assert.NotNil(t, event)
			return nil
		}, func(event *types.Event) error {
			executeCalled = true
			assert.NotNil(t, event)
			return fmt.Errorf("execution error")
		},
		"value-arg1", uint64(7531), false)
	assert.NotNil(t, err)
	assert.EqualError(t, err, "error executing handler: execution error")
	assert.True(t, validateCalled)
	assert.True(t, executeCalled)
}

// Test invalid event - no timestamp
func TestGoHandler_Execute_EventNoTimestamp(t *testing.T) {
	var validateCalled, executeCalled bool
	clearEnvironment()
	err := goHandlerExecuteUtil(t, &defaultHandlerConfig, "test/event-no-timestamp.json", defaultCmdLineArgs,
		func(event *types.Event) error {
			validateCalled = true
			assert.NotNil(t, event)
			return nil
		}, func(event *types.Event) error {
			executeCalled = true
			assert.NotNil(t, event)
			return nil
		},
		"value-arg1", uint64(7531), false)
	assert.NotNil(t, err)
	assert.EqualError(t, err, "timestamp is missing or must be greater than zero")
	assert.False(t, validateCalled)
	assert.False(t, executeCalled)
}

// Test invalid event - timestamp 0
func TestGoHandler_Execute_EventTimestampZero(t *testing.T) {
	var validateCalled, executeCalled bool
	clearEnvironment()
	err := goHandlerExecuteUtil(t, &defaultHandlerConfig, "test/event-timestamp-zero.json", defaultCmdLineArgs,
		func(event *types.Event) error {
			validateCalled = true
			assert.NotNil(t, event)
			return nil
		}, func(event *types.Event) error {
			executeCalled = true
			assert.NotNil(t, event)
			return nil
		},
		"value-arg1", uint64(7531), false)
	assert.NotNil(t, err)
	assert.EqualError(t, err, "timestamp is missing or must be greater than zero")
	assert.False(t, validateCalled)
	assert.False(t, executeCalled)
}

// Test invalid event - no entity
func TestGoHandler_Execute_EventNoEntity(t *testing.T) {
	var validateCalled, executeCalled bool
	clearEnvironment()
	err := goHandlerExecuteUtil(t, &defaultHandlerConfig, "test/event-no-entity.json", defaultCmdLineArgs,
		func(event *types.Event) error {
			validateCalled = true
			assert.NotNil(t, event)
			return nil
		}, func(event *types.Event) error {
			executeCalled = true
			assert.NotNil(t, event)
			return nil
		},
		"value-arg1", uint64(7531), false)
	assert.NotNil(t, err)
	assert.EqualError(t, err, "entity is missing from event")
	assert.False(t, validateCalled)
	assert.False(t, executeCalled)
}

// Test invalid event - invalid entity
func TestGoHandler_Execute_EventInvalidEntity(t *testing.T) {
	var validateCalled, executeCalled bool
	clearEnvironment()
	err := goHandlerExecuteUtil(t, &defaultHandlerConfig, "test/event-invalid-entity.json", defaultCmdLineArgs,
		func(event *types.Event) error {
			validateCalled = true
			assert.NotNil(t, event)
			return nil
		}, func(event *types.Event) error {
			executeCalled = true
			assert.NotNil(t, event)
			return nil
		},
		"value-arg1", uint64(7531), false)
	assert.NotNil(t, err)
	assert.EqualError(t, err, "entity name must not be empty")
	assert.False(t, validateCalled)
	assert.False(t, executeCalled)
}

// Test invalid event - no check
func TestGoHandler_Execute_EventNoCheck(t *testing.T) {
	var validateCalled, executeCalled bool
	clearEnvironment()
	err := goHandlerExecuteUtil(t, &defaultHandlerConfig, "test/event-no-check.json", defaultCmdLineArgs,
		func(event *types.Event) error {
			validateCalled = true
			assert.NotNil(t, event)
			return nil
		}, func(event *types.Event) error {
			executeCalled = true
			assert.NotNil(t, event)
			return nil
		},
		"value-arg1", uint64(7531), false)
	assert.NotNil(t, err)
	assert.EqualError(t, err, "check is missing from event")
	assert.False(t, validateCalled)
	assert.False(t, executeCalled)
}

// Test invalid event - invalid check
func TestGoHandler_Execute_EventInvalidCheck(t *testing.T) {
	var validateCalled, executeCalled bool
	clearEnvironment()
	err := goHandlerExecuteUtil(t, &defaultHandlerConfig, "test/event-invalid-check.json", defaultCmdLineArgs,
		func(event *types.Event) error {
			validateCalled = true
			assert.NotNil(t, event)
			return nil
		}, func(event *types.Event) error {
			executeCalled = true
			assert.NotNil(t, event)
			return nil
		},
		"value-arg1", uint64(7531), false)
	assert.NotNil(t, err)
	assert.EqualError(t, err, "check name must not be empty")
	assert.False(t, validateCalled)
	assert.False(t, executeCalled)
}

// Test unmarshalling error
func TestGoHandler_Execute_EventInvalidJson(t *testing.T) {
	var validateCalled, executeCalled bool
	clearEnvironment()
	err := goHandlerExecuteUtil(t, &defaultHandlerConfig, "test/event-invalid-json.json", defaultCmdLineArgs,
		func(event *types.Event) error {
			validateCalled = true
			assert.NotNil(t, event)
			return nil
		}, func(event *types.Event) error {
			executeCalled = true
			assert.NotNil(t, event)
			return nil
		},
		"value-arg1", uint64(7531), false)
	assert.NotNil(t, err)
	assert.EqualError(t, err, "Failed to unmarshal STDIN data: invalid character ':' after object key:value pair")
	assert.False(t, validateCalled)
	assert.False(t, executeCalled)
}

// Test fail to read stdin
func TestGoHandler_Execute_ReaderError(t *testing.T) {
	var validateCalled, executeCalled bool
	clearEnvironment()
	err := goHandlerExecuteUtil(t, &defaultHandlerConfig, "test/event-invalid-json.json", defaultCmdLineArgs,
		func(event *types.Event) error {
			validateCalled = true
			assert.NotNil(t, event)
			return nil
		}, func(event *types.Event) error {
			executeCalled = true
			assert.NotNil(t, event)
			return nil
		},
		"value-arg1", uint64(7531), false)
	assert.NotNil(t, err)
	assert.EqualError(t, err, "Failed to unmarshal STDIN data: invalid character ':' after object key:value pair")
	assert.False(t, validateCalled)
	assert.False(t, executeCalled)
}

// Test no keyspace
func TestGoHandler_Execute_NoKeyspace(t *testing.T) {
	var validateCalled, executeCalled bool
	clearEnvironment()
	handlerConfig := defaultHandlerConfig
	handlerConfig.Keyspace = ""
	err := goHandlerExecuteUtil(t, &handlerConfig, "test/event-check-entity-override.json", defaultCmdLineArgs,
		func(event *types.Event) error {
			validateCalled = true
			assert.NotNil(t, event)
			return nil
		}, func(event *types.Event) error {
			executeCalled = true
			assert.NotNil(t, event)
			return nil
		},
		"value-arg1", uint64(7531), false)
	assert.Nil(t, err)
	assert.True(t, validateCalled)
	assert.True(t, executeCalled)
}

func TestGoHandler_Execute_NoOptionValue(t *testing.T) {
	options := getDefaultOptions()
	handlerConfig := defaultHandlerConfig

	goHandler := NewGoHandler(&handlerConfig, options,
		func(event *types.Event) error {
			return nil
		}, func(event *types.Event) error {
			return nil
		})

	goHandler.cmdArgs.SetArgs(defaultCmdLineArgs)

	// Replace stdin reader with file reader
	goHandler.eventReader = getFileReader("test/event-check-entity-override.json")
	err := goHandler.Execute()

	assert.NotNil(t, err)
	assert.Errorf(t, err, "Option value must not be nil for option arg1")
}

func getFileReader(file string) io.Reader {
	reader, _ := os.Open(file)
	return reader
}

func clearEnvironment() {
	_ = os.Unsetenv("ENV_1")
	_ = os.Unsetenv("ENV_2")
	_ = os.Unsetenv("ENV_3")
}

func getDefaultOptions() []*HandlerConfigOption {
	option1 := defaultOption1
	option2 := defaultOption2
	option3 := defaultOption3
	return []*HandlerConfigOption{&option1, &option2, &option3}
}
