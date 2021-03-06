package common

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"os/user"
	"path"
	"reflect"
	"runtime"
	"strconv"
	"strings"
)

var (
	// Convenience Directories
	GoPath   = os.Getenv("GOPATH")
	ErisLtd  = path.Join(GoPath, "src", "github.com", "eris-ltd")
	usr, _   = user.Current() // error?!
	ErisRoot = ResolveErisRoot()

	// Major Directories
	ActionsPath        = path.Join(ErisRoot, "actions")
	BlockchainsPath    = path.Join(ErisRoot, "blockchains")
	DataContainersPath = path.Join(ErisRoot, "data")
	DappsPath          = path.Join(ErisRoot, "dapps")
	FilesPath          = path.Join(ErisRoot, "files")
	KeysPath           = path.Join(ErisRoot, "keys")
	LanguagesPath      = path.Join(ErisRoot, "languages")
	ServicesPath       = path.Join(ErisRoot, "services")
	ScratchPath        = path.Join(ErisRoot, "scratch")

	// Keys
	KeysDataPath = path.Join(KeysPath, "data")
	KeyNamesPath = path.Join(KeysPath, "names")

	// Scratch Directories (globally coordinated)
	EpmScratchPath  = path.Join(ScratchPath, "epm")
	LllcScratchPath = path.Join(ScratchPath, "lllc")
	SolcScratchPath = path.Join(ScratchPath, "sol")
	SerpScratchPath = path.Join(ScratchPath, "ser")

	// Blockchains stuff
	HEAD = path.Join(BlockchainsPath, "HEAD")
	Refs = path.Join(BlockchainsPath, "refs")
)

var MajorDirs = []string{
	ErisRoot, ActionsPath, BlockchainsPath, DataContainersPath, DappsPath, FilesPath, KeysPath, LanguagesPath, ServicesPath, KeysDataPath, KeyNamesPath, ScratchPath, EpmScratchPath, LllcScratchPath, SolcScratchPath, SerpScratchPath,
}

//---------------------------------------------
// user and process

func Usr() string {
	u, _ := user.Current()
	return u.HomeDir
}

func Exit(err error) {
	status := 0
	if err != nil {
		fmt.Println(err)
		status = 1
	}
	os.Exit(status)
}

func IfExit(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// user and process
//---------------------------------------------------------------------------
// filesystem

func AbsolutePath(Datadir string, filename string) string {
	if path.IsAbs(filename) {
		return filename
	}
	return path.Join(Datadir, filename)
}

func InitDataDir(Datadir string) error {
	_, err := os.Stat(Datadir)
	if err != nil {
		if os.IsNotExist(err) {
			err := os.MkdirAll(Datadir, 0777)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func ResolveErisRoot() string {
	var eris string
	if os.Getenv("ERIS") != "" {
		eris = os.Getenv("ERIS")
	} else {
		if runtime.GOOS == "windows" {
			home := os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
			if home == "" {
				home = os.Getenv("USERPROFILE")
			}
			eris = path.Join(home, ".eris")
		} else {
			eris = path.Join(Usr(), ".eris")
		}
	}
	return eris
}

// Create the default eris tree
func InitErisDir() (err error) {
	for _, d := range MajorDirs {
		err := InitDataDir(d)
		if err != nil {
			return err
		}
	}
	if _, err = os.Stat(HEAD); err != nil {
		_, err = os.Create(HEAD)
	}
	return
}

func ClearDir(dir string) error {
	fs, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, f := range fs {
		n := f.Name()
		if f.IsDir() {
			if err := os.RemoveAll(path.Join(dir, f.Name())); err != nil {
				return err
			}
		} else {
			if err := os.Remove(path.Join(dir, n)); err != nil {
				return err
			}
		}
	}
	return nil
}

func Copy(src, dst string) error {
	f, err := os.Stat(src)
	if err != nil {
		return err
	}
	if f.IsDir() {
		return copyDir(src, dst)
	}
	return copyFile(src, dst)
}

// assumes we've done our checking
func copyDir(src, dst string) error {
	fi, err := os.Stat(src)
	if err := os.MkdirAll(dst, fi.Mode()); err != nil {
		return err
	}
	fs, err := ioutil.ReadDir(src)
	if err != nil {
		return err
	}

	for _, f := range fs {
		s := path.Join(src, f.Name())
		d := path.Join(dst, f.Name())
		if f.IsDir() {
			if err := copyDir(s, d); err != nil {
				return err
			}
		} else {
			if err := copyFile(s, d); err != nil {
				return err
			}
		}
	}
	return nil
}

// common golang, really?
func copyFile(src, dst string) error {
	r, err := os.Open(src)
	if err != nil {
		return err
	}
	defer r.Close()

	w, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer w.Close()

	_, err = io.Copy(w, r)
	if err != nil {
		return err
	}
	return nil
}

// filesystem
//-------------------------------------------------------
// hex and ints

// keeps N bytes of the conversion
func NumberToBytes(num interface{}, N int) []byte {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.BigEndian, num)
	if err != nil {
		// TODO: get this guy a return error?
	}
	//fmt.Println("btyes!", buf.Bytes())
	if buf.Len() > N {
		return buf.Bytes()[buf.Len()-N:]
	}
	return buf.Bytes()
}

// s can be string, hex, or int.
// returns properly formatted 32byte hex value
func Coerce2Hex(s string) string {
	//fmt.Println("coercing to hex:", s)
	// is int?
	i, err := strconv.Atoi(s)
	if err == nil {
		return "0x" + hex.EncodeToString(NumberToBytes(int32(i), i/256+1))
	}
	// is already prefixed hex?
	if len(s) > 1 && s[:2] == "0x" {
		if len(s)%2 == 0 {
			return s
		}
		return "0x0" + s[2:]
	}
	// is unprefixed hex?
	if len(s) > 32 {
		return "0x" + s
	}
	pad := strings.Repeat("\x00", (32-len(s))) + s
	ret := "0x" + hex.EncodeToString([]byte(pad))
	//fmt.Println("result:", ret)
	return ret
}

func IsHex(s string) bool {
	if len(s) < 2 {
		return false
	}
	if s[:2] == "0x" {
		return true
	}
	return false
}

func AddHex(s string) string {
	if len(s) < 2 {
		return "0x" + s
	}

	if s[:2] != "0x" {
		return "0x" + s
	}

	return s
}

func StripHex(s string) string {
	if len(s) > 1 {
		if s[:2] == "0x" {
			s = s[2:]
			if len(s)%2 != 0 {
				s = "0" + s
			}
			return s
		}
	}
	return s
}

func StripZeros(s string) string {
	i := 0
	for ; i < len(s); i++ {
		if s[i] != '0' {
			break
		}
	}
	return s[i:]
}

// hex and ints
//---------------------------------------------------------------------------
// reflection and json

func WriteJson(config interface{}, config_file string) error {
	b, err := json.Marshal(config)
	if err != nil {
		return err
	}
	var out bytes.Buffer
	err = json.Indent(&out, b, "", "\t")
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(config_file, out.Bytes(), 0600)
	return err
}

func ReadJson(config interface{}, config_file string) error {
	b, err := ioutil.ReadFile(config_file)
	if err != nil {
		return err
	}
	err = json.Unmarshal(b, config)
	if err != nil {
		fmt.Println("error unmarshalling config from file:", err)
		return err
	}
	return nil
}

func NewInvalidKindErr(kind, k reflect.Kind) error {
	return fmt.Errorf("Invalid kind. Expected %s, received %s", kind, k)
}

func FieldFromTag(v reflect.Value, field string) (string, error) {
	iv := v.Interface()
	st := reflect.TypeOf(iv)
	for i := 0; i < v.NumField(); i++ {
		tag := st.Field(i).Tag.Get("json")
		if tag == field {
			return st.Field(i).Name, nil
		}
	}
	return "", fmt.Errorf("Invalid field name")
}

// Set a field in a struct value
// Field can be field name or json tag name
// Values can be strings that can be cast to int or bool
//  only handles strings, ints, bool
func SetProperty(cv reflect.Value, field string, value interface{}) error {
	f := cv.FieldByName(field)
	if !f.IsValid() {
		name, err := FieldFromTag(cv, field)
		if err != nil {
			return err
		}
		f = cv.FieldByName(name)
	}
	kind := f.Kind()

	k := reflect.ValueOf(value).Kind()
	if k != kind && k != reflect.String {
		return NewInvalidKindErr(kind, k)
	}

	if kind == reflect.String {
		f.SetString(value.(string))
	} else if kind == reflect.Int {
		if k != kind {
			v, err := strconv.Atoi(value.(string))
			if err != nil {
				return err
			}
			f.SetInt(int64(v))
		} else {
			f.SetInt(int64(value.(int)))
		}
	} else if kind == reflect.Bool {
		if k != kind {
			v, err := strconv.ParseBool(value.(string))
			if err != nil {
				return err
			}
			f.SetBool(v)
		} else {
			f.SetBool(value.(bool))
		}
	}
	return nil
}

// reflection and json
//---------------------------------------------------------------------------
// open text editors

func Editor(file string) error {
	editr := os.Getenv("EDITOR")
	if strings.Contains(editr, "/") {
		editr = path.Base(editr)
	}
	switch editr {
	case "", "vim", "vi":
		return vi(file)
	case "emacs":
		return emacs(file)
	}
	return fmt.Errorf("Unknown editor %s", editr)
}

func emacs(file string) error {
	cmd := exec.Command("emacs", file)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func vi(file string) error {
	cmd := exec.Command("vim", file)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
