package excel

import (
    "os"
    "strconv"
    "strings"
    "unsafe"
    "reflect"
    "fmt"
    "github.com/mattn/go-ole"
    "github.com/mattn/go-ole/oleutil"
)

type Option struct {
    Visible                      bool
    DisplayAlerts                bool
    ScreenUpdating               bool
}

type MSO struct {
    Option
    IuApp                      *ole.IUnknown
    IdExcel                    *ole.IDispatch
    IdWorkBooks                *ole.IDispatch
    Version                   float64
    FILEFORMAT          map[string]int
    FILEFORMAT11        map[string]string
}

type Sheet struct {
    IDisp *ole.IDispatch
}

//
type VARIANT ole.VARIANT

//convert MS VARIANT to string
func (va VARIANT) ToString() (ret string) {
    switch va.VT {
        case 2:
            v2:=(*int16)(unsafe.Pointer(&va.Val))
            ret = strconv.FormatInt(int64(*v2), 10)
        case 3:
            v3:=(*int32)(unsafe.Pointer(&va.Val))
            ret = strconv.FormatInt(int64(*v3), 10)
        case 4:
            v4:=(*float32)(unsafe.Pointer(&va.Val))
            ret = strconv.FormatFloat(float64(*v4), 'f', 2, 64)
        case 5:
            v5:=(*float64)(unsafe.Pointer(&va.Val))
            ret = strconv.FormatFloat(*v5, 'f', 2, 64)
        case 8:       //string
            v8:=(**uint16)(unsafe.Pointer(&va.Val))
            ret = ole.UTF16PtrToString(*v8)
        case 11:
            v11:=(*bool)(unsafe.Pointer(&va.Val))
            if *v11 {
                ret = "TRUE"
            } else {
                ret = "FALSE"
            }
        case 16:
            v16:=(*int8)(unsafe.Pointer(&va.Val))
            ret = strconv.FormatInt(int64(*v16), 10)
        case 17:
            v17:=(*uint8)(unsafe.Pointer(&va.Val))
            ret = strconv.FormatUint(uint64(*v17), 10)
        case 18:
            v18:=(*uint16)(unsafe.Pointer(&va.Val))
            ret = strconv.FormatUint(uint64(*v18), 10)
        case 19:
            v19:=(*uint32)(unsafe.Pointer(&va.Val))
            ret = strconv.FormatUint(uint64(*v19), 10)
        case 20:
            v20:=(*int64)(unsafe.Pointer(&va.Val))
            ret = strconv.FormatInt(int64(*v20), 10)
        case 21:
            v21:=(*uint64)(unsafe.Pointer(&va.Val))
            ret = strconv.FormatUint(uint64(*v21), 10)
    }
    return
}

//
func (option Option) Fields() (ret []string) {
    fields := reflect.Indirect(reflect.ValueOf(option)).Type()
    num := fields.NumField()
    for i:=0; i<num; i++ {
        ret = append(ret, fields.Field(i).Name)
    }
    return
}

//
func (mso *MSO) SetOption(option Option) {
    defer Except(0, "SetOption")
    elems := reflect.ValueOf(option)                      //.Elem()
    for _, key := range option.Fields() {
        val := elems.FieldByName(key).Bool()            //SetBool(true)
        oleutil.PutProperty(mso.IdExcel, key, val)
    }
}

//
func Init(options... Option) (mso *MSO) {
    ole.CoInitialize(0)
    app, _ := oleutil.CreateObject("Excel.Application")
    excel, _ := app.QueryInterface(ole.IID_IDispatch)
    wbs := oleutil.MustGetProperty(excel, "WorkBooks").ToIDispatch()
    ver, _ := strconv.ParseFloat(oleutil.MustGetProperty(excel, "Version").ToString(), 64)

    option := Option{Visible: true, DisplayAlerts: true, ScreenUpdating: true}
    if options != nil {
        option = options[0]
    }
    mso = &MSO{Option:option, IuApp:app, IdExcel:excel, IdWorkBooks:wbs, Version:ver}
    mso.SetOption(option)

    //XlFileFormat Enumeration: http://msdn.microsoft.com/en-us/library/office/ff198017%28v=office.15%29.aspx
    mso.FILEFORMAT = map[string]int {"txt":-4158, "csv":6, "html":44, "xlsx":51, "xls":56}
    mso.FILEFORMAT11 = map[string]string{"txt":"xlUnicodeText", "csv":"xlCSV", "html":"xlHTML", "xls":"xlNormal"}
    return
}

//
func New(options... Option) (mso *MSO) {
    defer Except(1, "New")
    mso = Init(options...)
    mso.WorkBookAdd()
    return
}

//
func Open(full string, options... Option) (mso *MSO) {
    defer Except(1, "Open")
    mso = Init(options...)
    mso.WorkBookOpen(full)
    return
}

//
func (mso *MSO) Save() {
    defer Except(0, "Save")
    for _, workbook := range mso.WorkBooks() {
        oleutil.MustCallMethod(workbook, "Save")
    }
}

//
func (mso *MSO) SaveAs(full string, args...string) {
    defer Except(0, "SaveAs")
    if true || mso.Version<=11.0 || args == nil {
        for _, workbook := range mso.WorkBooks() {
            oleutil.MustCallMethod(workbook, "SaveAs", full)
        }
    } else {
        val := args[0]
        ff := mso.FILEFORMAT[strings.ToLower(val)]
        for _, workbook := range mso.WorkBooks() {
            oleutil.MustCallMethod(workbook, "SaveAs", full, ff)
        }
    }
}

//
func (mso *MSO) Quit() {
    defer Except(0, "Quit")
    defer ole.CoUninitialize()
    oleutil.MustCallMethod(mso.IdWorkBooks, "Close")
    oleutil.MustCallMethod(mso.IdExcel, "Quit")
    mso.IdWorkBooks.Release()
    mso.IdExcel.Release()
    mso.IuApp.Release()
}

//
func (mso *MSO) Pick(workx string, id interface {}) (ret *ole.IDispatch) {
    defer Except(0, "mso.Pick")
    if id_int, ok := id.(int); ok {
        ret = oleutil.MustGetProperty(mso.IdExcel, workx, id_int).ToIDispatch()
    } else if id_str, ok := id.(string); ok {
        ret = oleutil.MustGetProperty(mso.IdExcel, workx, id_str).ToIDispatch()
    }
    return
}

//
func (mso *MSO) WorkBooksCount() (int) {
    return (int)(oleutil.MustGetProperty(mso.IdWorkBooks, "Count").Val)
}

//
func (mso *MSO) WorkBooks() (wbs []*ole.IDispatch) {
    num := mso.WorkBooksCount()
    for i:=1; i<=num; i++ {
        wbs = append(wbs, oleutil.MustGetProperty(mso.IdExcel, "WorkBooks", num).ToIDispatch())
    }
    return
}

//
func (mso *MSO) WorkBookAdd() (*ole.IDispatch) {
    return oleutil.MustCallMethod(mso.IdWorkBooks, "Add").ToIDispatch()
}

//
func (mso *MSO) WorkBookOpen(full string) (*ole.IDispatch) {
    defer Except(0, "WorkBookOpen")
    return oleutil.MustCallMethod(mso.IdWorkBooks, "open", full).ToIDispatch()
}

//
func (mso *MSO) WorkBookActivate(id interface {}) (wb *ole.IDispatch) {
    defer Except(0, "WorkBookActivate")
    wb = mso.Pick("WorkBooks", id)
    oleutil.MustCallMethod(wb, "Activate")
    return
}

//
func (mso *MSO) ActiveWorkBook() (*ole.IDispatch) {
    return oleutil.MustGetProperty(mso.IdExcel, "ActiveWorkBook").ToIDispatch()
}

//
func (mso *MSO) SheetsCount() (int) {
    sheets := oleutil.MustGetProperty(mso.IdExcel, "Sheets").ToIDispatch()
    defer sheets.Release()
    return (int)(oleutil.MustGetProperty(sheets, "Count").Val)
}

//
func (mso *MSO) Sheets() (sheets []Sheet) {
    num := mso.SheetsCount()
    for i:=1; i<=num; i++ {
        sheet := Sheet{oleutil.MustGetProperty(mso.IdExcel, "WorkSheets", i).ToIDispatch()}
        sheets = append(sheets, sheet)
    }
    return
}

//
func (mso *MSO) Sheet(id interface {}) (Sheet) {
    return Sheet{mso.Pick("WorkSheets", id)}
}

//
func (mso *MSO) SheetAdd(wb *ole.IDispatch, args... string) (sheet Sheet) {
    sheets := oleutil.MustGetProperty(wb, "Sheets").ToIDispatch()
    defer sheets.Release()
    sheet = Sheet{oleutil.MustCallMethod(sheets, "Add").ToIDispatch()}
    if args != nil {
        sheet.Name(args...)
    }
    sheet.Select()
    return
}

//
func (mso *MSO) SheetSelect(id interface {}) (sheet Sheet) {
    sheet = mso.Sheet(id)
    sheet.Select()
    return
}

//
func (sheet Sheet) Select() {
    oleutil.MustCallMethod(sheet.IDisp, "Select")
}

//
func (sheet Sheet) Name(args... string) (name string) {
    if args == nil {
        name = oleutil.MustGetProperty(sheet.IDisp, "Name").ToString()
    } else {
        name = args[0]
        oleutil.MustPutProperty(sheet.IDisp, "Name", name)
    }
    return
}

//
func (sheet Sheet) Cells(r int, c int, vals...interface{}) (ret string) {
    defer Except(0, "Cells")
    cell := oleutil.MustGetProperty(sheet.IDisp, "Range", Cell2r(r, c)).ToIDispatch()
    defer cell.Release()
    if vals == nil {
        //ret = oleutil.MustGetProperty(cell, "Value").ToString()
        ret = VARIANT(*oleutil.MustGetProperty(cell, "Value")).ToString()
    } else {
        val := vals[0]
        switch val.(type) {
            case string:
                oleutil.PutProperty(cell, "Value", val.(string))
            case int:
                oleutil.PutProperty(cell, "Value", val.(int))    //strconv.Itoa(val.(int)))
            case int32:
                oleutil.PutProperty(cell, "Value", val.(int32))    //, strconv.FormatInt(int64(val.(int32)), 0))
            case int64:
                oleutil.PutProperty(cell, "Value", val.(int64))    //, strconv.FormatInt(val.(int64), 0))
            case float32:
                oleutil.PutProperty(cell, "Value", val.(float32))    //, val.(float32))
            case float64:
                oleutil.PutProperty(cell, "Value", val.(float64))    //, val.(float64))
            case bool:
                oleutil.PutProperty(cell, "Value", val.(bool))    //, val.(bool))
           default:
                println("Cell not set: ", r, c)
        }
    }
    return
}

//(5,27) convert to "AA5"  // xp+office2003 cant use cell(1,1)
func Cell2r(x, y int) string {
    s := ""
    for y>0 {
        a, b := int(y/26), y%26
        if b==0 {
            a-=1
            b=26
        }
        s = string(rune(b+64))+s
        y = a
    }
    return s+strconv.Itoa(x)
}

//
func Except(exit int, info string) {
    r := recover()
    if r != nil {
        fmt.Println("Excel Except:", info, r)
        if exit>0 {
            os.Exit(exit)
        }
    }
}




