package gig

import (
	"errors"
	"fmt"
	"github.com/izuojian/gig/internal/utils"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

var (
	templatesLock sync.RWMutex

	gigTplFuncMap    = make(template.FuncMap)
	gigTemplates     = make(map[string]*template.Template)
	gigTplDelimLeft  = "{{"
	gigTplDelimRight = "}}"

	// gigTemplateExt stores the template extension which will build
	gigTemplateExt = []string{"tpl", "html", "gohtml"}

	gigTemplateEngines = map[string]templatePreProcessor{}
	gigTemplateFS      = defaultFSFunc
)

type templatePreProcessor func(root, path string, funcs template.FuncMap) (*template.Template, error)

type templateFile struct {
	root  string
	files map[string][]string
}

// AddFuncMap let user to register a func in the template.
func AddFuncMap(key string, fn interface{}) error {
	gigTplFuncMap[key] = fn
	return nil
}

func defaultFSFunc() http.FileSystem {
	return utils.FileSystem{}
}

// ExecuteViewPathTemplate applies the template with name and from specific viewPath to the specified data object,
// writing the output to wr.
// A template will be executed safely in parallel.
func ExecuteTemplate(wr io.Writer, name string, data interface{}) error {
	if t, ok := gigTemplates[name]; ok {
		var err error
		if t.Lookup(name) != nil {
			err = t.ExecuteTemplate(wr, name, data)
		} else {
			err = t.Execute(wr, data)
		}
		if err != nil {
			debugPrint("template Execute err: %v", err)
		}
		return err
	}
	panic("can't find templatefile in the path:" + name)
}

// visit will make the paths into two part,the first is subDir (without tf.root),the second is full path(without tf.root).
// if tf.root="views" and
// paths is "views/errors/404.html",the subDir will be "errors",the file will be "errors/404.html"
// paths is "views/admin/errors/404.html",the subDir will be "admin/errors",the file will be "admin/errors/404.html"
func (tf *templateFile) visit(paths string, f os.FileInfo, err error) error {
	if f == nil {
		return err
	}
	if f.IsDir() || (f.Mode()&os.ModeSymlink) > 0 {
		return nil
	}
	if !HasTemplateExt(paths) {
		return nil
	}

	replace := strings.NewReplacer("\\", "/")
	file := strings.TrimLeft(replace.Replace(paths[len(tf.root):]), "/")
	subDir := filepath.Dir(file)

	tf.files[subDir] = append(tf.files[subDir], file)
	return nil
}

// HasTemplateExt return this path contains supported template extension of gig or not.
func HasTemplateExt(paths string) bool {
	for _, v := range gigTemplateExt {
		if strings.HasSuffix(paths, "."+v) {
			return true
		}
	}
	return false
}

// BuildTemplate will build all template files in a directory.
// it makes gig can render any template file in view directory.
func LoadTemplates(dir string) error {
	var err error
	fs := gigTemplateFS()
	f, err := fs.Open(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return errors.New("dir open err")
	}
	defer f.Close()

	self := &templateFile{
		root:  dir,
		files: make(map[string][]string),
	}
	err = utils.Walk(fs, dir, func(path string, f os.FileInfo, err error) error {
		return self.visit(path, f, err)
	})
	if err != nil {
		fmt.Printf("Walk() returned %v\n", err)
		return err
	}

	for _, v := range self.files {
		for _, file := range v {
			templatesLock.Lock()
			ext := filepath.Ext(file)
			var t *template.Template
			if len(ext) == 0 {
				t, err = getTemplate(self.root, fs, file, v...)
			} else if fn, ok := gigTemplateEngines[ext[1:]]; ok {
				t, err = fn(self.root, file, gigTplFuncMap)
			} else {
				t, err = getTemplate(self.root, fs, file, v...)
			}
			if err != nil {
				debugPrint("parse template err:", file, err)
				templatesLock.Unlock()
				return err
			}
			gigTemplates[file] = t
			templatesLock.Unlock()
			if IsDebugging() {
				debugPrint("TPL: %4s", file)
			}
		}
	}
	return nil
}

func getTplDeep(root string, fs http.FileSystem, file string, parent string, t *template.Template) (*template.Template, [][]string, error) {
	var fileAbsPath string
	var rParent string
	var err error
	if strings.HasPrefix(file, "../") {
		rParent = filepath.Join(filepath.Dir(parent), file)
		fileAbsPath = filepath.Join(root, filepath.Dir(parent), file)
	} else {
		rParent = file
		fileAbsPath = filepath.Join(root, file)
	}
	f, err := fs.Open(fileAbsPath)
	if err != nil {
		panic("can't find template file:" + file)
	}
	defer f.Close()
	data, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, [][]string{}, err
	}
	t, err = t.New(file).Parse(string(data))
	if err != nil {
		return nil, [][]string{}, err
	}
	reg := regexp.MustCompile(gigTplDelimLeft + "[ ]*template[ ]+\"([^\"]+)\"")
	allSub := reg.FindAllStringSubmatch(string(data), -1)
	for _, m := range allSub {
		if len(m) == 2 {
			tl := t.Lookup(m[1])
			if tl != nil {
				continue
			}
			if !HasTemplateExt(m[1]) {
				continue
			}
			_, _, err = getTplDeep(root, fs, m[1], rParent, t)
			if err != nil {
				return nil, [][]string{}, err
			}
		}
	}
	return t, allSub, nil
}

func getTemplate(root string, fs http.FileSystem, file string, others ...string) (t *template.Template, err error) {
	t = template.New(file).Delims(gigTplDelimLeft, gigTplDelimRight).Funcs(gigTplFuncMap)
	var subMods [][]string
	t, subMods, err = getTplDeep(root, fs, file, "", t)
	if err != nil {
		return nil, err
	}
	t, err = _getTemplate(t, root, fs, subMods, others...)

	if err != nil {
		return nil, err
	}
	return
}

func _getTemplate(t0 *template.Template, root string, fs http.FileSystem, subMods [][]string, others ...string) (t *template.Template, err error) {
	t = t0
	for _, m := range subMods {
		if len(m) == 2 {
			tpl := t.Lookup(m[1])
			if tpl != nil {
				continue
			}
			//first check filename
			for _, otherFile := range others {
				if otherFile == m[1] {
					var subMods1 [][]string
					t, subMods1, err = getTplDeep(root, fs, otherFile, "", t)
					if err != nil {
						debugPrint("template parse file err:", err)
					} else if len(subMods1) > 0 {
						t, err = _getTemplate(t, root, fs, subMods1, others...)
					}
					break
				}
			}
			//second check define
			for _, otherFile := range others {
				var data []byte
				fileAbsPath := filepath.Join(root, otherFile)
				f, err := fs.Open(fileAbsPath)
				if err != nil {
					f.Close()
					debugPrint("template file parse error, not success open file:", err)
					continue
				}
				data, err = ioutil.ReadAll(f)
				f.Close()
				if err != nil {
					debugPrint("template file parse error, not success read file:", err)
					continue
				}
				reg := regexp.MustCompile(gigTplDelimLeft + "[ ]*define[ ]+\"([^\"]+)\"")
				allSub := reg.FindAllStringSubmatch(string(data), -1)
				for _, sub := range allSub {
					if len(sub) == 2 && sub[1] == m[1] {
						var subMods1 [][]string
						t, subMods1, err = getTplDeep(root, fs, otherFile, "", t)
						if err != nil {
							debugPrint("template parse file err:", err)
						} else if len(subMods1) > 0 {
							t, err = _getTemplate(t, root, fs, subMods1, others...)
							if err != nil {
								debugPrint("template parse file err:", err)
							}
						}
						break
					}
				}
			}
		}

	}
	return
}
