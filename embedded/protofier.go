// Copyright 2019 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package embedded

// dart-source: lib/src/embedded/protofier.dart

import (
	"fmt"

	"google.golang.org/protobuf/proto"

	"github.com/bancek/go-sass/orderedmap"
	"github.com/bancek/go-sass/value"
)

// Protofier translates Sass values to and from their protobuf encodings
// within one custom-function call.
//
// The dispatcher serves the host-function callbacks created while
// deprotofying; functions and mixins assign the opaque IDs that let the
// host hand compiler-side functions and mixins back in later calls.
// Argument lists protofied here are retained by ID so the host can report
// which ones it inspected.
//
// Matches Dart: class Protofier
type Protofier struct {
	dispatcher    *CompilationDispatcher
	functions     *OpaqueRegistry
	mixins        *OpaqueRegistry
	argumentLists []*value.SassArgumentList
}

// NewProtofier returns a Protofier that sends host-function callbacks
// through dispatcher and tracks compiler function and mixin IDs in
// functions and mixins. Each call gets a fresh Protofier because the
// argument-list IDs it hands out are only meaningful for that call.
//
// Matches Dart: Protofier constructor
func NewProtofier(dispatcher *CompilationDispatcher, functions, mixins *OpaqueRegistry) *Protofier {
	return &Protofier{
		dispatcher: dispatcher,
		functions:  functions,
		mixins:     mixins,
	}
}

// Protofy converts a Sass value to its protocol buffer encoding.
//
// Strings keep their quoted flag; numbers split into a value plus numerator
// and denominator unit lists; colors carry the space name with absent
// channels left nil so the host can tell missing apart from zero; argument
// lists are retained under a one-based ID (zero is reserved for
// host-created lists) with keywords protofied without marking them
// accessed; plain lists keep separator and brackets; maps encode each entry
// in order; calculations recurse through protofyCalculation;
// compiler-side functions and mixins travel as their opaque registry IDs;
// booleans and null become singletons.
//
// Matches Dart: protofy
func (p *Protofier) Protofy(v value.Value) (*Value, error) {
	result := &Value{}
	switch val := v.(type) {
	case *value.SassString:
		result.Value = &Value_String_{String_: &Value_String{
			Text:   val.Text,
			Quoted: val.HasQuotes,
		}}
	case value.SassNumber:
		result.Value = &Value_Number_{Number: p.protofyNumber(val)}
	case *value.SassColor:
		result.Value = &Value_Color_{Color: &Value_Color{
			Space:    val.Space().Name(),
			Channel1: val.Channel0OrNil(),
			Channel2: val.Channel1OrNil(),
			Channel3: val.Channel2OrNil(),
			Alpha:    val.AlphaOrNil(),
		}}
	case *value.SassArgumentList:
		p.argumentLists = append(p.argumentLists, val)
		contents, _ := val.AsList()
		sep, err := p.protofySeparator(val.Separator())
		if err != nil {
			return nil, err
		}
		args := &Value_ArgumentList{
			Id:        uint32(len(p.argumentLists)),
			Separator: sep,
		}
		contentsProto, err := p.protofyValues(contents)
		if err != nil {
			return nil, err
		}
		args.Contents = contentsProto
		args.Keywords = make(map[string]*Value)
		for k, kwv := range val.KeywordsWithoutMarking().Entries() {
			kwProto, err := p.Protofy(kwv)
			if err != nil {
				return nil, err
			}
			args.Keywords[k] = kwProto
		}
		result.Value = &Value_ArgumentList_{ArgumentList: args}
	case *value.SassList:
		contents, _ := val.AsList()
		sep, err := p.protofySeparator(val.Separator())
		if err != nil {
			return nil, err
		}
		contentsProto, err := p.protofyValues(contents)
		if err != nil {
			return nil, err
		}
		result.Value = &Value_List_{List: &Value_List{
			Separator:   sep,
			HasBrackets: val.HasBrackets(),
			Contents:    contentsProto,
		}}
	case *value.SassMap:
		entries := &Value_Map{}
		for k, kv := range val.Entries() {
			kProto, err := p.Protofy(k)
			if err != nil {
				return nil, err
			}
			vProto, err := p.Protofy(kv)
			if err != nil {
				return nil, err
			}
			entries.Entries = append(entries.Entries, &Value_Map_Entry{
				Key:   kProto,
				Value: vProto,
			})
		}
		result.Value = &Value_Map_{Map: entries}
	case *value.SassCalculation:
		calc, err := p.protofyCalculation(val)
		if err != nil {
			return nil, err
		}
		result.Value = &Value_Calculation_{Calculation: calc}
	case *value.SassFunction:
		result.Value = &Value_CompilerFunction_{CompilerFunction: &Value_CompilerFunction{
			Id: uint32(p.functions.GetID(val)),
		}}
	case *value.SassMixin:
		result.Value = &Value_CompilerMixin_{CompilerMixin: &Value_CompilerMixin{
			Id: uint32(p.mixins.GetID(val)),
		}}
	case *value.SassBoolean:
		if val.Value {
			result.Value = &Value_Singleton{Singleton: SingletonValue_TRUE}
		} else {
			result.Value = &Value_Singleton{Singleton: SingletonValue_FALSE}
		}
	case *value.SassNull:
		result.Value = &Value_Singleton{Singleton: SingletonValue_NULL}
	default:
		return nil, fmt.Errorf("Unknown Value %v", v)
	}
	return result, nil
}

// protofyValues protofies each value in order, failing the whole slice on
// the first value the protocol cannot represent.
func (p *Protofier) protofyValues(vals []value.Value) ([]*Value, error) {
	result := make([]*Value, len(vals))
	for i, v := range vals {
		protoVal, err := p.Protofy(v)
		if err != nil {
			return nil, err
		}
		result[i] = protoVal
	}
	return result, nil
}

// DeprotofyResponse converts a host function-call response into a Sass
// value.
//
// Before decoding the return value it marks every argument list the host
// reports as accessed, so later keyword-access tracking observes the host's
// reads. Unknown IDs are parameter errors.
//
// Matches Dart: deprotofyResponse
func (p *Protofier) DeprotofyResponse(response *InboundMessage_FunctionCallResponse) (value.Value, error) {
	for _, id := range response.GetAccessedArgumentLists() {
		argList, err := p.argumentListForID(id)
		if err != nil {
			return nil, err
		}
		argList.Keywords() // Mark keywords accessed, mirroring Dart's property read.
	}
	return p.deprotofy(response.GetSuccess())
}

// deprotofy decodes one protobuf value into its Sass representation.
//
// An empty-text string still round-trips its quoted flag; colors resolve
// their space by name and treat absent channels as missing rather than
// zero; an argument list carrying a nonzero ID resolves to the retained
// list instead of decoding inline; undecided separators are rejected once
// a list holds more than one element; empty lists and maps decode to their
// canonical empty values; compiler functions and mixins must name a known
// registry ID; host functions are rewrapped as callbacks through the
// dispatcher; interpolations decode parenthesized and unquoted; anything
// else is a missing-field or unknown-singleton protocol error.
// Out-of-range channel data surfaces as a parameter error naming the
// offending channel.
//
// Matches Dart: _deprotofy
func (p *Protofier) deprotofy(v *Value) (value.Value, error) {
	switch v.GetValue().(type) {
	case *Value_String_:
		s := v.GetString_()
		if s.GetText() == "" {
			quoted := s.GetQuoted()
			return value.EmptySassString(&quoted), nil
		}
		return &value.SassString{Text: s.GetText(), HasQuotes: s.GetQuoted()}, nil

	case *Value_Number_:
		return p.deprotofyNumber(v.GetNumber()), nil

	case *Value_Color_:
		c := v.GetColor()
		space, err := value.ColorSpaceFromName(c.GetSpace(), nil)
		if err != nil {
			return nil, paramsError(err.Error())
		}
		var channel1, channel2, channel3, alpha *float64
		if c.Channel1 != nil {
			channel1 = proto.Float64(c.GetChannel1())
		}
		if c.Channel2 != nil {
			channel2 = proto.Float64(c.GetChannel2())
		}
		if c.Channel3 != nil {
			channel3 = proto.Float64(c.GetChannel3())
		}
		if c.Alpha != nil {
			alpha = proto.Float64(c.GetAlpha())
		}
		result, err := value.NewColorForSpaceInternal(space, channel1, channel2, channel3, alpha)
		if err != nil {
			return nil, paramsError(err.Error())
		}
		return result, nil

	case *Value_ArgumentList_:
		al := v.GetArgumentList()
		if al.GetId() != 0 {
			return p.argumentListForID(al.GetId())
		}

		separator, err := p.deprotofySeparator(al.GetSeparator())
		if err != nil {
			return nil, err
		}
		length := len(al.GetContents())
		if separator == value.ListSeparatorUndecided && length > 1 {
			return nil, paramsError(fmt.Sprintf(
				"List %v can't have an undecided separator because it has %d elements", v, length))
		}

		contents := make([]value.Value, length)
		for i, item := range al.GetContents() {
			val, err := p.deprotofy(item)
			if err != nil {
				return nil, err
			}
			contents[i] = val
		}
		keywords := orderedmap.New[string, value.Value]()
		for k, vv := range al.GetKeywords() {
			val, err := p.deprotofy(vv)
			if err != nil {
				return nil, err
			}
			keywords.Put(k, val)
		}
		argList, err := value.NewSassArgumentList(contents, keywords, separator)
		if err != nil {
			return nil, err
		}
		return argList, nil

	case *Value_List_:
		l := v.GetList()
		separator, err := p.deprotofySeparator(l.GetSeparator())
		if err != nil {
			return nil, err
		}
		if len(l.GetContents()) == 0 {
			brackets := l.GetHasBrackets()
			return value.NewSassListEmpty(&separator, &brackets), nil
		}

		length := len(l.GetContents())
		if separator == value.ListSeparatorUndecided && length > 1 {
			return nil, paramsError(fmt.Sprintf(
				"List %v can't have an undecided separator because it has %d elements", v, length))
		}

		var contents []value.Value
		for _, item := range l.GetContents() {
			val, err := p.deprotofy(item)
			if err != nil {
				return nil, err
			}
			contents = append(contents, val)
		}
		lst, err := value.NewSassList(contents, separator, l.GetHasBrackets())
		if err != nil {
			return nil, paramsError(err.Error())
		}
		return lst, nil

	case *Value_Map_:
		m := v.GetMap()
		if len(m.GetEntries()) == 0 {
			return value.EmptySassMap(), nil
		}
		contents := make(map[value.Value]value.Value, len(m.GetEntries()))
		for _, entry := range m.GetEntries() {
			k, err := p.deprotofy(entry.GetKey())
			if err != nil {
				return nil, err
			}
			kv, err := p.deprotofy(entry.GetValue())
			if err != nil {
				return nil, err
			}
			contents[k] = kv
		}
		return value.NewSassMap(contents), nil

	case *Value_CompilerFunction_:
		id := int(v.GetCompilerFunction().GetId())
		if fn := p.functions.Get(id); fn != nil {
			if sf, ok := fn.(*value.SassFunction); ok {
				return sf, nil
			}
		}
		return nil, paramsError(fmt.Sprintf("CompilerFunction.id %d doesn't match any known functions", id))

	case *Value_HostFunction_:
		hf := v.GetHostFunction()
		hfID := hf.GetId()
		callable, err := hostCallable(
			p.dispatcher, p.functions, p.mixins, hf.GetSignature(), &hfID)
		if err != nil {
			return nil, err
		}
		return value.NewSassFunction(callable), nil

	case *Value_CompilerMixin_:
		id := int(v.GetCompilerMixin().GetId())
		if m := p.mixins.Get(id); m != nil {
			if sm, ok := m.(*value.SassMixin); ok {
				return sm, nil
			}
		}
		return nil, paramsError(fmt.Sprintf("CompilerMixin.id %d doesn't match any known mixins", id))

	case *Value_Calculation_:
		return p.deprotofyCalculation(v.GetCalculation())

	case *Value_Singleton:
		switch v.GetSingleton() {
		case SingletonValue_TRUE:
			return value.SassTrue, nil
		case SingletonValue_FALSE:
			return value.SassFalse, nil
		case SingletonValue_NULL:
			return value.Null, nil
		default:
			return nil, fmt.Errorf("Unknown Value.singleton %v", v.GetSingleton())
		}

	default:
		return nil, mandatoryError("Value.value")
	}
}

// protofyNumber splits n into its numeric value plus numerator and
// denominator unit lists, the wire form of a Sass number.
//
// Matches Dart: _protofyNumber
func (p *Protofier) protofyNumber(n value.SassNumber) *Value_Number {
	return &Value_Number{
		Value:        n.NumValue(),
		Numerators:   n.NumNumeratorUnits(),
		Denominators: n.NumDenominatorUnits(),
	}
}

// deprotofyNumber rebuilds a Sass number from its wire value and unit
// lists. Numbers need no validation here: any value and unit combination
// is representable.
//
// Matches Dart: _deprotofyNumber
func (p *Protofier) deprotofyNumber(n *Value_Number) value.SassNumber {
	return value.SassNumberWithUnits(n.GetValue(), n.GetNumerators(), n.GetDenominators())
}

// protofySeparator maps a list separator onto its wire enum. The undecided
// separator round-trips as its own value; an unrecognized separator is an
// internal error, since it can only come from a corrupt Sass value rather
// than from host input.
//
// Matches Dart: _protofySeparator
func (p *Protofier) protofySeparator(sep value.ListSeparator) (ListSeparator, error) {
	switch sep {
	case value.ListSeparatorComma:
		return ListSeparator_COMMA, nil
	case value.ListSeparatorSpace:
		return ListSeparator_SPACE, nil
	case value.ListSeparatorSlash:
		return ListSeparator_SLASH, nil
	case value.ListSeparatorUndecided:
		return ListSeparator_UNDECIDED, nil
	default:
		return 0, fmt.Errorf("Unknown ListSeparator %v", sep)
	}
}

// deprotofySeparator maps a wire separator enum back to a Sass separator.
// An unrecognized wire value is an internal error, since the proto schema
// constrains what the host may send.
//
// Matches Dart: _deprotofySeparator
func (p *Protofier) deprotofySeparator(sep ListSeparator) (value.ListSeparator, error) {
	switch sep {
	case ListSeparator_COMMA:
		return value.ListSeparatorComma, nil
	case ListSeparator_SPACE:
		return value.ListSeparatorSpace, nil
	case ListSeparator_SLASH:
		return value.ListSeparatorSlash, nil
	case ListSeparator_UNDECIDED:
		return value.ListSeparatorUndecided, nil
	default:
		return 0, fmt.Errorf("Unknown ListSeparator %v", sep)
	}
}

// protofyCalculation encodes a calculation's name and arguments in order;
// each argument recurses through protofyCalculationValue so nested numbers,
// calculations, strings, and operations keep their shape.
//
// Matches Dart: _protofyCalculation
func (p *Protofier) protofyCalculation(c *value.SassCalculation) (*Value_Calculation, error) {
	result := &Value_Calculation{
		Name: c.Name,
	}
	for _, arg := range c.Arguments {
		cv, err := p.protofyCalculationValue(arg)
		if err != nil {
			return nil, err
		}
		result.Arguments = append(result.Arguments, cv)
	}
	return result, nil
}

// protofyCalculationValue encodes one value nested inside a calculation:
// a number keeps its units, a nested calculation recurses, a string keeps
// its text (calculations never carry quotes on the wire), and an operation
// encodes its operator plus recursively encoded operands.
//
// Matches Dart: _protofyCalculationValue
func (p *Protofier) protofyCalculationValue(arg any) (*Value_Calculation_CalculationValue, error) {
	result := &Value_Calculation_CalculationValue{}
	switch v := arg.(type) {
	case value.SassNumber:
		result.Value = &Value_Calculation_CalculationValue_Number{Number: p.protofyNumber(v)}
	case *value.SassCalculation:
		calc, err := p.protofyCalculation(v)
		if err != nil {
			return nil, err
		}
		result.Value = &Value_Calculation_CalculationValue_Calculation{Calculation: calc}
	case *value.SassString:
		result.Value = &Value_Calculation_CalculationValue_String_{String_: v.Text}
	case *value.CalculationOperation:
		op, err := p.protofyCalculationOperator(v.Operator)
		if err != nil {
			return nil, err
		}
		left, err := p.protofyCalculationValue(v.Left)
		if err != nil {
			return nil, err
		}
		right, err := p.protofyCalculationValue(v.Right)
		if err != nil {
			return nil, err
		}
		result.Value = &Value_Calculation_CalculationValue_Operation{Operation: &Value_Calculation_CalculationOperation{
			Operator: op,
			Left:     left,
			Right:    right,
		}}
	default:
		return nil, fmt.Errorf("Unknown calculation value %v", arg)
	}
	return result, nil
}

// protofyCalculationOperator maps a calculation operator onto its wire
// enum. An unrecognized operator is an internal error, since it can only
// come from a corrupt Sass value rather than from host input.
//
// Matches Dart: _protofyCalculationOperator
func (p *Protofier) protofyCalculationOperator(op value.CalculationOperator) (CalculationOperator, error) {
	switch op {
	case value.CalculationOperatorPlus:
		return CalculationOperator_PLUS, nil
	case value.CalculationOperatorMinus:
		return CalculationOperator_MINUS, nil
	case value.CalculationOperatorTimes:
		return CalculationOperator_TIMES, nil
	case value.CalculationOperatorDividedBy:
		return CalculationOperator_DIVIDE, nil
	default:
		return 0, fmt.Errorf("Unknown CalculationOperator %v", op)
	}
}

// deprotofyCalculation decodes a wire calculation into the matching Sass
// constructor, enforcing each function's arity on the wire: calc takes
// exactly one argument, clamp takes one to three (absent middle and max
// stay nil), and min and max each take at least one. An unrecognized name
// is a parameter error naming the offending calculation.
//
// Matches Dart: _deprotofyCalculation
func (p *Protofier) deprotofyCalculation(c *Value_Calculation) (value.Value, error) {
	switch c.GetName() {
	case "calc":
		if len(c.GetArguments()) != 1 {
			return nil, paramsError("Value.Calculation.arguments must have exactly one argument for calc().")
		}
		arg, err := p.deprotofyCalculationValue(c.GetArguments()[0])
		if err != nil {
			return nil, err
		}
		result, err := value.NewCalc(arg)
		if err != nil {
			return nil, err
		}
		return result, nil

	case "clamp":
		args := c.GetArguments()
		if len(args) < 1 || len(args) > 3 {
			return nil, paramsError("Value.Calculation.arguments must have 1 to 3 arguments for clamp().")
		}
		var min, val, max any
		min, err := p.deprotofyCalculationValue(args[0])
		if err != nil {
			return nil, err
		}
		if len(args) > 1 {
			val, err = p.deprotofyCalculationValue(args[1])
			if err != nil {
				return nil, err
			}
		}
		if len(args) > 2 {
			max, err = p.deprotofyCalculationValue(args[2])
			if err != nil {
				return nil, err
			}
		}
		result, err := value.NewClamp(min, val, max)
		if err != nil {
			return nil, err
		}
		return result, nil

	case "min":
		if len(c.GetArguments()) == 0 {
			return nil, paramsError("Value.Calculation.arguments must have at least 1 argument for min().")
		}
		var args []any
		for _, a := range c.GetArguments() {
			arg, err := p.deprotofyCalculationValue(a)
			if err != nil {
				return nil, err
			}
			args = append(args, arg)
		}
		result, err := value.NewMin(args...)
		if err != nil {
			return nil, err
		}
		return result, nil

	case "max":
		if len(c.GetArguments()) == 0 {
			return nil, paramsError("Value.Calculation.arguments must have at least 1 argument for max().")
		}
		var args []any
		for _, a := range c.GetArguments() {
			arg, err := p.deprotofyCalculationValue(a)
			if err != nil {
				return nil, err
			}
			args = append(args, arg)
		}
		result, err := value.NewMax(args...)
		if err != nil {
			return nil, err
		}
		return result, nil

	default:
		return nil, paramsError(fmt.Sprintf(
			`Value.Calculation.name "%s" is not a recognized calculation type.`, c.GetName()))
	}
}

// deprotofyCalculationValue decodes one wire value nested inside a
// calculation: numbers and nested calculations recurse, strings decode
// unquoted, interpolations decode parenthesized and unquoted so they
// survive the round trip, and operations rebuild through Operate. An unset
// value is a missing-field protocol error.
//
// Matches Dart: _deprotofyCalculationValue
func (p *Protofier) deprotofyCalculationValue(cv *Value_Calculation_CalculationValue) (any, error) {
	switch cv.GetValue().(type) {
	case *Value_Calculation_CalculationValue_Number:
		return p.deprotofyNumber(cv.GetNumber()), nil
	case *Value_Calculation_CalculationValue_Calculation:
		return p.deprotofyCalculation(cv.GetCalculation())
	case *Value_Calculation_CalculationValue_String_:
		return &value.SassString{Text: cv.GetString_(), HasQuotes: false}, nil
	case *Value_Calculation_CalculationValue_Interpolation:
		return &value.SassString{Text: "(" + cv.GetInterpolation() + ")", HasQuotes: false}, nil
	case *Value_Calculation_CalculationValue_Operation:
		op := cv.GetOperation()
		calcOp, err := p.deprotofyCalculationOperator(op.GetOperator())
		if err != nil {
			return nil, err
		}
		left, err := p.deprotofyCalculationValue(op.GetLeft())
		if err != nil {
			return nil, err
		}
		right, err := p.deprotofyCalculationValue(op.GetRight())
		if err != nil {
			return nil, err
		}
		return value.Operate(calcOp, left, right)
	default:
		return nil, mandatoryError("Value.Calculation.value")
	}
}

// deprotofyCalculationOperator maps a wire calculation operator back to
// its Sass form. An unrecognized wire value is an internal error, since
// the proto schema constrains what the host may send.
//
// Matches Dart: _deprotofyCalculationOperator
func (p *Protofier) deprotofyCalculationOperator(op CalculationOperator) (value.CalculationOperator, error) {
	switch op {
	case CalculationOperator_PLUS:
		return value.CalculationOperatorPlus, nil
	case CalculationOperator_MINUS:
		return value.CalculationOperatorMinus, nil
	case CalculationOperator_TIMES:
		return value.CalculationOperatorTimes, nil
	case CalculationOperator_DIVIDE:
		return value.CalculationOperatorDividedBy, nil
	default:
		return 0, fmt.Errorf("Unknown CalculationOperator %v", op)
	}
}

// argumentListForID resolves a one-based wire argument-list ID to the
// retained SassArgumentList. Zero is reserved for host-created lists and
// can never name a retained one, so IDs below one and IDs past the end of
// the retained slice are both parameter errors.
//
// Matches Dart: _argumentListForId
func (p *Protofier) argumentListForID(id uint32) (*value.SassArgumentList, error) {
	if id < 1 {
		return nil, paramsError(fmt.Sprintf("Value.ArgumentList.id %d can't be marked as accessed", id))
	}
	if int(id) > len(p.argumentLists) {
		return nil, paramsError(fmt.Sprintf(
			"Value.ArgumentList.id %d doesn't match any known argument lists", id))
	}
	return p.argumentLists[int(id)-1], nil
}
