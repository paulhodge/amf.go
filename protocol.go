package amf

import (
        "encoding/binary"
        "fmt"
        "os"
        "reflect"
        )

type Reader interface {
    Read(p []byte) (n int, err os.Error)
}

type Writer interface {
    Write(p []byte) (n int, err os.Error)
}

// Read an AMF3 value from the stream.
func ReadValueAmf3(stream Reader) (interface{}, os.Error) {
    cxt := DecodeContext{}
    cxt.amfVersion = 3
    result := readValueAmf3(stream, &cxt)
    return result, cxt.decodeError
}

// Type markers
const (
    amf0_numberType = 0
    amf0_booleanType = 1
    amf0_stringType = 2
    amf0_objectType = 3
    amf0_movieClipType = 4
    amf0_nullType = 5
    amf0_undefinedType = 6
    amf0_referenceType = 7
    amf0_ecmaArrayType = 8
    amf0_objectEndType = 9
    amf0_strictArrayType = 10
    amf0_dateType = 11
    amf0_longStringType = 12
    amf0_unsupporedType = 13
    amf0_recordsetType = 14
    amf0_xmlObjectType = 15
    amf0_typedObjectType = 16
    amf0_avmPlusObjectType = 17

    amf3_undefinedType = 0
    amf3_nullType = 1
    amf3_falseType = 2
    amf3_trueType = 3
    amf3_integerType = 4
    amf3_doubleType = 5
    amf3_stringType = 6
    amf3_xmlType = 7
    amf3_dateType = 8
    amf3_arrayType = 9
    amf3_objectType = 10
    amf3_avmPlusXmlType = 11
    amf3_byteArrayType = 12
)

type DecodeContext struct {
    amfVersion uint16

    stringTable []string
    classTable []*AvmClass
    objectTable []ObjectReference

    decodeError os.Error
}

// Objects in the object table can either be AvmObjects or arrays
type ObjectReference struct {
    asAvmObject *AvmObject
    asArray *AvmArray
}

func (cxt *DecodeContext) useAmf3() bool {
    return cxt.amfVersion == 3
}
func (cxt *DecodeContext) saveError(err os.Error) {
    if err == nil {
        return
    }
    if cxt.decodeError != nil {
        fmt.Println("warning: duplicate errors on DecodeContext")
    } else {
        cxt.decodeError = err
    }
}
func (cxt *DecodeContext) storeObjectInTable(obj *AvmObject) {
    cxt.objectTable = append(cxt.objectTable, ObjectReference{obj, nil})
}
func (cxt *DecodeContext) storeArrayInTable(array *AvmArray) {
    cxt.objectTable = append(cxt.objectTable, ObjectReference{nil, array})
}

type MessageBundle struct {
    amfVersion uint16
    headers []AmfHeader
    messages []AmfMessage
}

type AmfHeader struct {
    name string
    mustUnderstand bool
    value AmfValue
}
type AmfMessage struct {
    targetUri string
    responseUri string
    args []AmfValue
    body AmfValue
}

// AmfValue is a variant type.
type AmfValue interface {}

type AvmObject struct {
    class *AvmClass
    fields map[string] AmfValue
}

type AvmClass struct {
    name string
    externalizable bool
    dynamic bool
    properties []string
}

// An "Array" in AVM land is actually stored as a combination of an array and
// a dictionary.
type AvmArray struct {
    elements []AmfValue
    fields map[string]AmfValue
}

// Helper functions.
func readByte(stream Reader) (uint8, os.Error) {
    buf := make([]byte, 1)
    _, err := stream.Read(buf)
    return buf[0], err
}

func readUint8(stream Reader) (uint8,os.Error) {
    var value uint8
    err := binary.Read(stream, binary.BigEndian, &value)
    return value, err
}
func readUint16(stream Reader) (uint16,os.Error) {
    var value uint16
    err := binary.Read(stream, binary.BigEndian, &value)
    return value, err
}
func writeUint16(stream Writer, value uint16) {
    binary.Write(stream, binary.BigEndian, &value)
}
func readUint32(stream Reader) uint32 {
    var value uint32
    binary.Read(stream, binary.BigEndian, &value)
    return value
}
func writeUint32(stream Writer, value uint32) {
    binary.Write(stream, binary.BigEndian, &value)
}
func readFloat64(stream Reader) (float64,os.Error) {
    var value float64
    err := binary.Read(stream, binary.BigEndian, &value)
    return value, err
}
func WriteFloat64(stream Writer, value float64) os.Error {
    return binary.Write(stream, binary.BigEndian, &value)
}
func ReadString(stream Reader) (string,os.Error) {
    length,err := readUint16(stream)
    if err != nil {
        return "", err
    }
    data := make([]byte, length)
    _,err = stream.Read(data)
    return string(data), err
}
func readStringLength(stream Reader, length int) (string,os.Error) {
    data := make([]byte, length)
    n,err := stream.Read(data)
    if n < length {
        return "", os.NewError(fmt.Sprintf(
            "Not enough bytes in readStringLength (expected %d, found %d)", length, n))
    }
    return string(data), err
}
func WriteString(stream Writer, str string) os.Error {
    binary.Write(stream, binary.BigEndian, uint16(len(str)))
    _,err := stream.Write([]byte(str))
    return err
}
func writeByte(stream Writer, b uint8) os.Error {
    return binary.Write(stream, binary.BigEndian, b)
}
func writeBool(stream Writer, b bool) {
    val := 0x0
    if b {
        val = 0xff
    }
    binary.Write(stream, binary.BigEndian, uint8(val))
}

func WriteInt32(stream Writer, val int32) os.Error {
    return binary.Write(stream, binary.BigEndian, val)
}

// Read a 29-bit compact encoded integer (as defined in AVM3)
func readUint29(stream Reader) (uint32, os.Error) {
    var result uint32 = 0
    for i := 0; i < 4; i++ {
        b, err := readByte(stream)

        if err != nil {
            return result, err
        }

        if i == 3 {
            // Last byte does not use the special 0x80 bit.
            result = (result << 8) + uint32(b)
        } else {
            result = (result << 7) + (uint32(b) & 0x7f)
        }

        if (b & 0x80) == 0 {
            break
        }
    }
    return result, nil
}

func WriteUint29(stream Writer, value uint32) os.Error {

    // Make sure the value is only 29 bits.
    remainder := value & 0x1fffffff
    if remainder != value {
        fmt.Println("warning: WriteUint29 received a value that does not fit in 29 bits")
    }

    // Peel off the value 7 bits at a time.
    for i := 0; i < 4; i++ {
        if i == 3 {
            // Last byte does not use the special 0x80 bit.
            writeByte(stream, uint8(remainder & 0xff))
            break
        } else {
            byteOut := uint8(remainder & 0x7f)
            remainder = remainder >> 7

            // Set the 0x80 bit if there is still more data to come.
            if remainder != 0 {
                byteOut = byteOut | 0x80
            }

            writeByte(stream, byteOut)

            if remainder == 0 {
                break
            }
        }
    }

    return nil
}

func readStringAmf3(stream Reader, cxt *DecodeContext) string {
    ref,err := readUint29(stream)

    if err != nil {
        cxt.saveError(err)
        return ""
    }

    // Check the low bit to see if this is a reference
    if (ref & 1) == 0 {
        index := int(ref >> 1)
        if index >= len(cxt.stringTable) {
            cxt.saveError(os.NewError(fmt.Sprintf("String reference out of range: %d", index)))
            return ""
        }

        return cxt.stringTable[index]
    }

    length := int(ref >> 1)

    if (length == 0) {
        return ""
    }

    str,err := readStringLength(stream, length)
    cxt.saveError(err)
    cxt.stringTable = append(cxt.stringTable, str)

    return str
}

func WriteStringAmf3(stream Writer, s string) {
    length := len(s)

    // TODO: Support outgoing string references.

    WriteUint29(stream, uint32((length << 1) + 1))

    stream.Write([]byte(s))
}

func readObject3(stream Reader, cxt *DecodeContext) *AvmObject {

    ref,_ := readUint29(stream)

    // Check the low bit to see if this is a reference
    if (ref & 1) == 0 {
        fmt.Printf("Looking up a object ref: %d\n", int(ref>>1))
        index := int(ref >> 1)
        objRef := cxt.objectTable[index]
        if objRef.asAvmObject == nil {
            err := os.NewError(fmt.Sprintf(
                "Tried to read reference %d as an AvmObject, but stored object "+
                "has diffent type\n", index))
            cxt.saveError(err)
            return nil
        }
        return objRef.asAvmObject
    }

    class := readClassDefinitionAmf3(stream, ref, cxt)

    object := AvmObject{}
    object.class = class
    object.fields = make(map[string] AmfValue)

    // Store the object in the table before doing any decoding.
    cxt.storeObjectInTable(&object)

    // Read static fields
    for _,name := range class.properties {
        value := readValueAmf3(stream, cxt)
        object.fields[name] = value
    }

    if class.dynamic {
        // Parse dynamic fields
        for {
            name := readStringAmf3(stream, cxt)
            if name == "" {
                break
            }

            value := readValueAmf3(stream, cxt)
            object.fields[name] = value
        }
    }

    return &object
}

func writeObject3(stream Writer, value interface{}) os.Error {

    fmt.Printf("writeObject3 attempting to write a value of type %s\n",
        reflect.ValueOf(value).Type().Name())

    return nil
}

func writeAvmObject3(stream Writer, value *AvmObject) os.Error {
    // TODO: Support outgoing object references.

    // writeClassDefinitionAmf3 will also write the ref section.
    writeClassDefinitionAmf3(stream, value.class)

    return nil
}

func readClassDefinitionAmf3(stream Reader, ref uint32, cxt *DecodeContext) *AvmClass {
    // Check for a reference to an existing class definition
    if (ref & 2) == 0 {
        return cxt.classTable[int(ref >> 2)]
    }

    // Parse a class definition
    className := readStringAmf3(stream, cxt)

    externalizable := ref & 4 != 0
    dynamic := ref & 8 != 0
    propertyCount := ref >> 4

    class := AvmClass{className, externalizable, dynamic, make([]string, propertyCount)}

    // Property names
    for i := uint32(0); i < propertyCount; i++ {
        class.properties[i] = readStringAmf3(stream, cxt)
    }

    // Save the new class in the loopup table
    cxt.classTable = append(cxt.classTable, &class)

    return &class
}

func writeClassDefinitionAmf3(stream Writer, class *AvmClass) {
    // TODO: Support class references
    var ref uint32 = 0

    ref += 0x2

    if class.externalizable {
        ref += 0x4
    }
    if class.dynamic {
        ref += 0x8
    }

    ref += uint32(len(class.properties) << 4)

    WriteUint29(stream, ref)

    WriteStringAmf3(stream, class.name)

    // Property names
    for _,name := range class.properties {
        WriteStringAmf3(stream, name)
    }
}

func readArray3(stream Reader, cxt *DecodeContext) *AvmArray {
    ref,_ := readUint29(stream)

    // Check the low bit to see if this is a reference
    if (ref & 1) == 0 {
        index := int(ref >> 1)
        objRef := cxt.objectTable[index]
        if objRef.asArray == nil {
            err := os.NewError(fmt.Sprintf(
                "Tried to read reference %d as an AvmArray, but stored object "+
                "has diffent type\n", index))
            cxt.saveError(err)
            return nil
        }
        return objRef.asArray
    }

    elementCount := int(ref >> 1)

    result := &AvmArray{}

    // Store the object in the table before doing any decoding.
    cxt.storeArrayInTable(result)

    // Read name-value pairs, if any.
    key := readStringAmf3(stream, cxt)

    if key != "" {
        result.fields = make(map[string]AmfValue)
        for key != "" {
            result.fields[key] = readValueAmf3(stream, cxt)
            key = readStringAmf3(stream, cxt)
        }
    }

    // Read dense elements
    result.elements = make([]AmfValue, elementCount)
    for i := 0; i < elementCount; i++ {
        result.elements[i] = readValueAmf3(stream, cxt)
    }

    return result
}

func writeFlatArray3(stream Writer, value []interface{}) os.Error {
    elementCount := len(value)

    // TODO: Support outgoing array references
    ref := (elementCount << 1) + 1

    WriteUint29(stream, uint32(ref))

    // Write an empty key since this is just a flat array.
    WriteStringAmf3(stream, "")

    // Write dense elements
    for i := 0; i < elementCount; i++ {
        WriteValueAmf3(stream, value[i])
    }
    return nil
}

func writeMixedArray3(stream Writer, value *AvmArray) os.Error {
    elementCount := len(value.elements)

    // TODO: Support outgoing array references
    ref := (elementCount << 1) + 1

    WriteUint29(stream, uint32(ref))

    // Write fields
    for k,v := range value.fields {
        WriteStringAmf3(stream, k)
        WriteValueAmf3(stream, v)
    }

    // Write a null name to indicate the end of fields.
    WriteStringAmf3(stream, "")

    // Write dense elements
    for i := 0; i < elementCount; i++ {
        WriteValueAmf3(stream, value.elements[i])
    }
    return nil
}

func readValue(stream Reader, cxt *DecodeContext) AmfValue {
    if cxt.amfVersion == 0 {
        return readValueAmf0(stream, cxt)
    }

    return readValueAmf3(stream, cxt)
}

func readValueAmf0(stream Reader, cxt *DecodeContext) AmfValue {

    typeMarker,_ := readByte(stream)

    // Most AMF0 types are not yet supported.

    // Type markers
    switch typeMarker {
    case amf0_numberType:
        val,err := readFloat64(stream)
        cxt.saveError(err)
        return val
    case amf0_booleanType:
        val,err := readUint8(stream)
        cxt.saveError(err)
        return val != 0
    case amf0_stringType:
        str,err := ReadString(stream)
        cxt.saveError(err)
        return str
    case amf0_objectType:
        result := map[string]interface{} {}
        for true {
            c1,_ := readByte(stream)
            c2,_ := readByte(stream)
            length := int(c1) << 8 + int(c2)
            name,_ := readStringLength(stream, length)
            result[name] = readValueAmf0(stream, cxt)
        }
        return result

    case amf0_movieClipType:
        fmt.Printf("movie clip type not supported")
    case amf0_nullType:
        return nil
    case amf0_undefinedType:
        return nil
    case amf0_referenceType:
    case amf0_ecmaArrayType:
    case amf0_objectEndType:
    case amf0_strictArrayType:
    case amf0_dateType:
    case amf0_longStringType:
    case amf0_unsupporedType:
    case amf0_recordsetType:
    case amf0_xmlObjectType:
    case amf0_typedObjectType:
    case amf0_avmPlusObjectType:
        return readValueAmf3(stream, cxt)
    }

    fmt.Printf("AMF0 type marker was not supported: %d", typeMarker)
    return nil
}

func readValueAmf3(stream Reader, cxt *DecodeContext) AmfValue {

    // Read type marker
    typeMarker,err := readByte(stream)

    if err != nil {
        cxt.saveError(err)
        return nil
    }

    // Flash Player 9 will sometimes wrap data as an AMF0 value, which just means that
    // there might be an additional type code (amf0_avmPlusObjectType), which we can
    // unambiguously ignore here.

    if typeMarker == amf0_avmPlusObjectType {
        typeMarker,_ = readByte(stream)
    }

    switch typeMarker {
    case amf3_nullType, amf3_undefinedType:
        return nil
    case amf3_falseType:
        return false
    case amf3_trueType:
        return true
    case amf3_integerType:
        result,err := readUint29(stream)
        cxt.saveError(err)
        return result
    case amf3_doubleType:
        result,err := readFloat64(stream)
        cxt.saveError(err)
        return result
    case amf3_stringType:
        return readStringAmf3(stream, cxt)
    case amf3_objectType:
        return readObject3(stream, cxt)
    case amf3_arrayType:
        return readArray3(stream, cxt)
    default:
        cxt.saveError(os.NewError("AMF3 type marker was not supported"))
        return nil
    }
    return nil
}

func WriteValueAmf3(stream Writer, value interface{}) os.Error {

    switch t := value.(type) {
    case string:
        writeByte(stream, amf3_stringType)
        str,_ := value.(string)
        WriteStringAmf3(stream, str)
    case int, int32, uint32:
        n,_ := value.(uint32)
        writeByte(stream, amf3_integerType)
        return WriteUint29(stream, n)
    case float32:
        n,_ := value.(float32)
        writeByte(stream, amf3_doubleType)
        return WriteFloat64(stream, float64(n))
    case float64:
        n,_ := value.(float64)
        writeByte(stream, amf3_doubleType)
        return WriteFloat64(stream, n)
    case bool:
        if value == false {
            return writeByte(stream, amf3_falseType)
        } else {
            return writeByte(stream, amf3_trueType)
        }
    case nil:
        return writeByte(stream, amf3_nullType)
    case []interface{}:
        writeByte(stream, amf3_arrayType)
        arr,_ := value.([]interface{})
        return writeFlatArray3(stream, arr)
    case *AvmArray:
        arr,_ := value.(*AvmArray)
        writeByte(stream, amf3_arrayType)
        return writeMixedArray3(stream, arr)
    case *AvmObject:
        writeByte(stream, amf3_objectType)
        obj,_ := value.(*AvmObject)
        writeAvmObject3(stream, obj)
    default:
        fmt.Printf("WriteValueAmf3 didn't recognize type: %s\n", reflect.ValueOf(value).Type().Name())
    }

    return nil
}

