package dir

import (
	"boxroom/resource/immobile"
	"boxroom/resource/tree"
	globleimmobile "boxroom/util/globle-immobile"
	utilfunc "boxroom/util/util-func"
	utillog "boxroom/util/util-log"
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"path/filepath"
	"sigs.k8s.io/yaml"
	"time"
)

type StorageDirDefinition interface {
	TransformResourceTreeToLocal(root *tree.KubernetesRoot) (map[string]string, string, error)
	TransformLocalToResourceTree(localStorageDir, localFile string, root *tree.KubernetesRoot) error
	GetAssistLogLocalDir(root *tree.KubernetesRoot) (string, string, string)
	GetAssistLogRemoteDir(root *tree.KubernetesRoot) (string, string)
	GetAssistLogLocalZipDir(localLogName string, localStatusLogName string) (string, string, error)
	GetRemoteStoragePrefixAndDelimiter(root *tree.KubernetesRoot) (string, string)
	GetDownLoadDirMap(root *tree.KubernetesRoot) (map[string]string, string, string, error)
	ParseCommonPrefix(prefix string) string
	PreHandleRestoreRoot(root *tree.KubernetesRoot)
}

var log = new(utillog.NewLog).GetLogger()

type DefaultStorageDirDefinition struct {
}

func (dir *DefaultStorageDirDefinition) PreHandleRestoreRoot(root *tree.KubernetesRoot) {
	if root.TreeKind == immobile.TreeRestoreKind {
		root.TreeName = immobile.TreeRestoreKind + time.Now().Format(globleimmobile.TimestampFormat) + "-" + root.TreeName
	}
}

func (dir *DefaultStorageDirDefinition) TransformResourceTreeToLocal(root *tree.KubernetesRoot) (map[string]string, string, error) {

	fileOperator := utilfunc.NewWorkDirFileOperator()
	dirOperator := utilfunc.NewWorkDirOperator()

	if exist := fileOperator.CheckDirIsValid(globleimmobile.WorkDir); !exist {
		err := fileOperator.CreateDir(globleimmobile.WorkDir)
		if err != nil {
			log.Error(err)
			return nil, "", err
		}
	}
	if len(root.TreeName) == 0 {
		nowTime := time.Now().Format(globleimmobile.TimestampFormat)
		root.TreeName = root.TreeKind + nowTime
	}

	localWorkDir, localStorageDir := getLocalStorageDir(root)
	remoteStorageDir := getRemoteStorageDir(root)

	err := createS3GroupDir(dirOperator.GenerateDirPath(localStorageDir, JsonStorageKind), root, JsonStorageKind)
	if err != nil {
		log.Error(err)
		return nil, "", err
	}
	err = createS3GroupDir(dirOperator.GenerateDirPath(localStorageDir, YamlStorageKind), root, YamlStorageKind)
	if err != nil {
		log.Error(err)
		return nil, "", err
	}

	storageMap, err := createStorageMap(localStorageDir, remoteStorageDir)
	if err != nil {
		log.Error(err)
		return nil, "", err
	}

	return storageMap, localWorkDir, nil
}

func (dir *DefaultStorageDirDefinition) TransformLocalToResourceTree(localStorageDir, localFile string, root *tree.KubernetesRoot) error {
	err := utilfunc.NewTgzPacker().UnPack(localFile, localStorageDir)
	if err != nil {
		log.Error(err)
		return err
	}

	fileOperator := utilfunc.NewWorkDirFileOperator()

	err = fileOperator.DeleteDirOrFile(localFile)
	if err != nil {
		log.Error(err)
		return err
	}

	resourceMap, err := getLocalResourceMap(localStorageDir)
	if err != nil {
		log.Error(err)
		return err
	}

	err = buildResourceTree(root, resourceMap)
	if err != nil {
		log.Error(err)
		return err
	}

	return nil
}

func (dir *DefaultStorageDirDefinition) GetAssistLogLocalDir(root *tree.KubernetesRoot) (string, string, string) {
	dirOperator := utilfunc.NewWorkDirOperator()

	localWorkDir := getLocalWorkDir()
	localLoggerName, localStatusLoggerName := getLoggerFileName(root)

	localLoggerDir := dirOperator.GenerateDirPath(localWorkDir, localLoggerName)

	localStatusLoggerDir := dirOperator.GenerateDirPath(localWorkDir, localStatusLoggerName)

	return localWorkDir, localLoggerDir, localStatusLoggerDir
}
func (dir *DefaultStorageDirDefinition) GetAssistLogRemoteDir(root *tree.KubernetesRoot) (string, string) {
	dirOperator := utilfunc.NewWorkDirOperator()

	localLoggerName, localStatusLoggerName := getLoggerFileName(root)

	loggerRemoteDir := dirOperator.GenerateDirPath(getRemoteStorageDir(root), dirOperator.GenerateObjectName(localLoggerName, UploadStorageKind))

	statusLoggerRemoteDir := dirOperator.GenerateDirPath(getRemoteStorageDir(root), dirOperator.GenerateObjectName(localStatusLoggerName, UploadStorageKind))

	return loggerRemoteDir, statusLoggerRemoteDir
}

func (dir *DefaultStorageDirDefinition) GetAssistLogLocalZipDir(localLogDir string, localStatusLogDir string) (string, string, error) {
	dirOperator := utilfunc.WorkDirOperator{}
	tgzPacker := utilfunc.NewTgzPacker()

	tgzLogDir := dirOperator.GenerateObjectName(localLogDir, UploadStorageKind)
	tgzStatusLogDir := dirOperator.GenerateObjectName(localStatusLogDir, UploadStorageKind)

	err := tgzPacker.Pack(localLogDir, tgzLogDir)
	if err != nil {
		return "", "", err
	}

	err = tgzPacker.Pack(localStatusLogDir, tgzStatusLogDir)
	if err != nil {
		return "", "", err
	}

	return tgzLogDir, tgzStatusLogDir, nil
}

func (dir *DefaultStorageDirDefinition) GetRemoteStoragePrefixAndDelimiter(root *tree.KubernetesRoot) (string, string) {
	dirOperator := utilfunc.NewWorkDirOperator()
	return dirOperator.GenerateDirPath(root.Name, root.TreeKind) + "/", "/"
}

func (dir *DefaultStorageDirDefinition) GetDownLoadDirMap(root *tree.KubernetesRoot) (map[string]string, string, string, error) {
	dirOperator := utilfunc.NewWorkDirOperator()

	remoteStorageDir := getRemoteStorageDir(root)
	workDir, localStorageDir := getLocalStorageDir(root)

	remoteFile := dirOperator.GenerateDirPath(remoteStorageDir, dirOperator.GenerateObjectName(JsonStorageKind, UploadStorageKind))
	localFile := dirOperator.GenerateDirPath(localStorageDir, dirOperator.GenerateObjectName(JsonStorageKind, UploadStorageKind))

	downloadDirMap := map[string]string{}
	downloadDirMap[remoteFile] = localFile

	return downloadDirMap, workDir, localStorageDir, nil
}

func (dir *DefaultStorageDirDefinition) ParseCommonPrefix(prefix string) string {
	return filepath.Base(prefix)
}

func buildResourceTree(root *tree.KubernetesRoot, resourceMap map[string]map[string][]byte) error {
	dirOperator := utilfunc.NewWorkDirOperator()

	root.Groups = map[string]*tree.Group{}

	for _, resource := range resourceMap {
		metadata := &tree.ObjectMetadata{}
		err := metadata.CovertJsonToStruct(resource[ObjectMetadataFileName])
		if err != nil {
			log.Error(err)
			return err
		}

		object := root.AddChildren(metadata.Group).
			AddChildren(metadata.Version).
			AddChildren(metadata.Resource, metadata.IsCluster).
			AddChildren(metadata.Namespace).
			AddChildren(metadata.Name)

		err = object.CovertJsonToStruct(resource[dirOperator.GenerateObjectName(metadata.Name, JsonStorageKind)])
		if err != nil {
			log.Error(err)
			return err
		}
		object.Metadata = metadata
		object.GVR = &schema.GroupVersionResource{
			Group:    metadata.Group,
			Version:  metadata.Version,
			Resource: metadata.Resource,
		}
	}

	return nil
}

func getLocalResourceMap(localStorageDir string) (map[string]map[string][]byte, error) {

	fileOperator := utilfunc.NewWorkDirFileOperator()

	resourceMap := map[string]map[string][]byte{}

	err := filepath.WalkDir(localStorageDir, func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}

		dirName, fileName := filepath.Split(path)

		reader, err := fileOperator.OpenFile(path)
		if err != nil {
			log.Error(err)
			return err
		}
		buff := bytes.Buffer{}
		_, err = buff.ReadFrom(reader)
		if err != nil {
			log.Error(err)
			return err
		}

		if _, ok := resourceMap[dirName]; !ok {
			resourceMap[dirName] = map[string][]byte{}
		}

		resourceMap[dirName][fileName] = buff.Bytes()

		defer func() {
			err = reader.Close()
			if err != nil {
				log.Error(err)
			}
		}()
		return nil
	})

	if err != nil {
		log.Error(err)
		return nil, err
	}

	return resourceMap, nil
}

func getLocalWorkDir() string {
	dirOperator := utilfunc.NewWorkDirOperator()
	randomStr := utilfunc.RandStr(30)
	return dirOperator.GenerateDirPath(globleimmobile.WorkDir, randomStr)
}

func getLocalStorageDir(root *tree.KubernetesRoot) (string, string) {
	dirOperator := utilfunc.NewWorkDirOperator()
	workDir := getLocalWorkDir()
	localStorageDir := dirOperator.GenerateDirPath(workDir, root.Name, root.TreeKind, root.TreeName)
	return workDir, localStorageDir
}

func getRemoteStorageDir(root *tree.KubernetesRoot) string {
	dirOperator := utilfunc.NewWorkDirOperator()
	return dirOperator.GenerateDirPath(root.Name, root.TreeKind, root.TreeName)
}

func createStorageMap(localStorageDir, remoteStorageDir string) (map[string]string, error) {
	dirOperator := utilfunc.NewWorkDirOperator()
	fileOperator := utilfunc.NewWorkDirFileOperator()

	jsonLocalPath := dirOperator.GenerateDirPath(localStorageDir, JsonStorageKind)
	jsonLocalFile := dirOperator.GenerateObjectName(jsonLocalPath, UploadStorageKind)
	jsonRemoteFile := dirOperator.GenerateDirPath(remoteStorageDir, dirOperator.GenerateObjectName(JsonStorageKind, UploadStorageKind))

	yamlLocalPath := dirOperator.GenerateDirPath(localStorageDir, YamlStorageKind)
	yamlLocalFile := dirOperator.GenerateObjectName(yamlLocalPath, UploadStorageKind)
	yamlRemoteFile := dirOperator.GenerateDirPath(remoteStorageDir, dirOperator.GenerateObjectName(YamlStorageKind, UploadStorageKind))

	err := utilfunc.NewTgzPacker().Pack(jsonLocalPath, jsonLocalFile)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	err = utilfunc.NewTgzPacker().Pack(yamlLocalPath, yamlLocalFile)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	if !(fileOperator.CheckFileIsValid(jsonLocalFile) && fileOperator.CheckFileIsValid(yamlLocalFile)) {
		e := fmt.Sprintf("there is no upload file in this localPath: %s", localStorageDir)
		log.Error(e)
		return nil, errors.New(e)
	}

	storageMap := map[string]string{}
	storageMap[jsonLocalFile] = jsonRemoteFile
	storageMap[yamlLocalFile] = yamlRemoteFile

	defer func() {
		err := fileOperator.DeleteDirOrFile(jsonLocalPath)
		err = fileOperator.DeleteDirOrFile(yamlLocalPath)
		if err != nil {
			log.Error(err)
		}
	}()

	return storageMap, nil
}

func createS3GroupDir(parentsDir string, root *tree.KubernetesRoot, storageKind string) error {
	dirOperator := utilfunc.NewWorkDirOperator()
	for _, group := range root.Groups {
		err := createS3VersionDir(dirOperator.GenerateDirPath(parentsDir, group.Name+"_"), group, storageKind)
		if err != nil {
			log.Error(err)
			return err
		}
	}
	return nil
}

func createS3VersionDir(parentsDir string, group *tree.Group, storageKind string) error {
	for _, version := range group.Versions {
		err := createS3ResourceDir(parentsDir+version.Name, version, storageKind)
		if err != nil {
			log.Error(err)
			return err
		}
	}
	return nil
}

func createS3ResourceDir(parentsDir string, version *tree.Version, storageKind string) error {
	dirOperator := utilfunc.NewWorkDirOperator()
	for _, resource := range version.Resources {
		err := createS3NamespaceDir(dirOperator.GenerateDirPath(parentsDir, resource.Name), resource, storageKind)
		if err != nil {
			log.Error(err)
			return err
		}
	}
	return nil
}

func createS3NamespaceDir(parentsDir string, resource *tree.Resource, storageKind string) error {
	dirOperator := utilfunc.NewWorkDirOperator()
	for _, namespace := range resource.Namespaces {
		err := createObjectDir(dirOperator.GenerateDirPath(parentsDir, namespace.Name), namespace, storageKind)
		if err != nil {
			log.Error(err)
			return err
		}
	}
	return nil
}

func createObjectDir(parentsDir string, namespace *tree.Namespace, storageKind string) error {
	for _, object := range namespace.Objects {
		err := storeObjectLocal(parentsDir, object, storageKind)
		if err != nil {
			log.Error(err)
			return err
		}
	}
	return nil
}

func storeObjectLocal(parentsDir string, object *tree.Object, storageKind string) error {
	dirOperator := utilfunc.NewWorkDirOperator()
	fileOperator := utilfunc.NewWorkDirFileOperator()
	objectDir := dirOperator.GenerateDirPath(parentsDir, object.Name)
	err := fileOperator.CreateDir(objectDir)
	if err != nil {
		log.Error(err)
		return err
	}
	objectFileName := dirOperator.GenerateObjectName(dirOperator.GenerateDirPath(objectDir, object.Name), storageKind)
	metadataFileName := dirOperator.GenerateDirPath(objectDir, ObjectMetadataFileName)
	objectFile, err := fileOperator.CreateFile(objectFileName)
	defer func() {
		err = objectFile.Close()
		if err != nil {
			log.Error(err)
		}
	}()
	if err != nil {
		log.Error(err)
		return err
	}
	metadataFile, err := fileOperator.CreateFile(metadataFileName)
	defer func() {
		err = metadataFile.Close()
		if err != nil {
			log.Error(err)
		}
	}()
	if err != nil {
		log.Error(err)
		return err
	}
	jsonDefinition, err := object.CovertStructToJson()
	if err != nil {
		log.Error(err)
		return err
	}
	metadataDefinition, err := object.Metadata.CovertStructToJson()
	if err != nil {
		log.Error(err)
		return err
	}
	switch storageKind {
	case JsonStorageKind:
		err = fileOperator.WriteFile(objectFile, jsonDefinition)
		if err != nil {
			log.Error(err)
			return err
		}
	case YamlStorageKind:
		yamlDefinition, err := yaml.JSONToYAML(jsonDefinition)
		if err != nil {
			log.Error(err)
			return err
		}
		err = fileOperator.WriteFile(objectFile, yamlDefinition)
		if err != nil {
			log.Error(err)
			return err
		}
	}
	err = fileOperator.WriteFile(metadataFile, metadataDefinition)
	if err != nil {
		log.Error(err)
		return err
	}
	return nil
}

func getLoggerFileName(root *tree.KubernetesRoot) (string, string) {
	dirOperator := utilfunc.NewWorkDirOperator()
	return dirOperator.GenerateObjectName(root.TreeName, "log"), dirOperator.GenerateObjectName(root.TreeName+"-"+"status", "log")
}
