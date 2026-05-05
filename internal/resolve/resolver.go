package resolve

import (
	"fmt"
	"strconv"

	descriptor "github.com/jlkendrick/grimoire/internal/descriptor"
)

func ResolveFunctionDescriptor(fd *descriptor.FunctionDescriptor) error {
	// For now, we can just simply cast the default values to the correct type
	// Can do more sophisticated type resolution later
	for i, param := range fd.Params {
		// Later we can try to resolve the type from the default value
		if param.Default == nil || param.ResolvedType == nil {
			continue
		}
		switch param.ResolvedType.Name {
		case "string", "str":
			fd.Params[i].Default = param.Default.(string)

		case "int":
			int_default, err := strconv.Atoi(param.Default.(string))
			if err != nil {
				return fmt.Errorf("error converting default value to int: %v", err)
			}
			fd.Params[i].Default = int_default

		case "bool":
			bool_default, err := strconv.ParseBool(param.Default.(string))
			if err != nil {
				return fmt.Errorf("error converting default value to bool: %v", err)
			}
			fd.Params[i].Default = bool_default

		case "float":
			float_default, err := strconv.ParseFloat(param.Default.(string), 64)
			if err != nil {
				return fmt.Errorf("error converting default value to float: %v", err)
			}
			fd.Params[i].Default = float_default

		default:
			return fmt.Errorf("unsupported type: %s", param.ResolvedType.Name)
		}
	}

	return nil
}