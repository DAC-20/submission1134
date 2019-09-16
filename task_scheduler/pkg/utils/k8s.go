package utils

import (
	"io/ioutil"
	"os"
	"log"
	"strings"
	"encoding/json"
	"strconv"
	"time"
	
	"github.com/comail/colog"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	apps_v1beta1 "k8s.io/api/apps/v1beta1"
	yaml2 "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// init k8s client
func InitClient() (clientset *kubernetes.Clientset, err error) {
	var (
		restConf *rest.Config
	)

	if restConf, err = getRestConf(); err != nil {
		return
	}

	// generate clientset
	if clientset, err = kubernetes.NewForConfig(restConf); err != nil {
		goto END
	}
END:
	return
}


// generate k8s restful client
func getRestConf() (restConf *rest.Config, err error) {
	var (
		kubeconfig []byte
	)

	// read kubeconfig from file
	configfile := os.Getenv("HOME") + "/.kube/config"
	// if kubeconfig, err = ioutil.ReadFile("./admin.conf"); err != nil {
	if kubeconfig, err = ioutil.ReadFile(configfile); err != nil {
		goto END
	}
	// generate rest client
	if restConf, err = clientcmd.RESTConfigFromKubeConfig(kubeconfig); err != nil {
		goto END
	}
END:
	return
}

func InitLog(level string) {
	logLevel := stringToLevel(level)
	colog.SetDefaultLevel(logLevel)
	colog.SetMinLevel(logLevel)
	colog.SetFormatter(&colog.StdFormatter{
		Colors: true,
		Flag:   log.Ldate | log.Ltime | log.Lshortfile,
	})
	colog.Register()
}

func stringToLevel(level string) colog.Level {
	switch level := strings.ToUpper(level); level {
	case "TRACE":
		return colog.LTrace
	case "DEBUG":
		return colog.LDebug
	case "INFO":
		return colog.LInfo
	case "WARNING":
		return colog.LWarning
	case "ERROR":
		return colog.LError
	case "ALERT":
		return colog.LAlert
	default:
		log.Printf("warning: LOG_LEVEL=\"%s\" is empty or invalid, fallling back to \"INFO\".\n", level)
		return colog.LInfo
	}
}

func YamlToDeployment(file string) (*apps_v1beta1.Deployment, error) {
	var (
		deployJson []byte
		deployment  = apps_v1beta1.Deployment{}
		err error
	)
	if deployJson, err = yamlToJson(file); err != nil {
		return nil, err
	}
	// json to deployment struct
	if err = json.Unmarshal(deployJson, &deployment); err != nil {
		return nil, err
	}
	return &deployment, nil	
}

func YamlToPod(file string) (*v1.Pod, error) {
	var (
		deployJson []byte
		pod  = v1.Pod{}
		err error
	)
	if deployJson, err = yamlToJson(file); err != nil {
		return nil, err
	}
	// json to pod struct
	if err = json.Unmarshal(deployJson, &pod); err != nil {
		return nil, err
	}
	return &pod, nil	
}

func yamlToJson(file string) ([]byte, error) {
	var (	
		deployYaml []byte
		deployJson []byte
		err error
	)
	// read YAML file
	if deployYaml, err = ioutil.ReadFile(file); err != nil {
		return nil, err
	}	
	// yaml to json
	if deployJson, err = yaml2.ToJSON(deployYaml); err != nil {
		return nil, err
	}
	return deployJson, nil
}

func ApplyDeployment(clientset *kubernetes.Clientset, deployment *apps_v1beta1.Deployment) error {
	var err error
	if _, err = clientset.AppsV1beta1().Deployments("default").Get(deployment.Name, metav1.GetOptions{}); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		// create one if not exist
		if _, err = clientset.AppsV1beta1().Deployments("default").Create(deployment); err != nil {
			return err
		}
		log.Println("debug: deployment not found, created")
	} else {	 // delete, then create
		if err = clientset.AppsV1beta1().Deployments("default").Delete(deployment.Name, &metav1.DeleteOptions{}); err != nil {
			log.Println("warning: delete deployment error:", err)
			return err
		} else {
			time.Sleep(2 * time.Second)
			if _, err = clientset.AppsV1beta1().Deployments("default").Create(deployment); err != nil {
				return err
			}
		}
		log.Println("debug: deployment found, deleted and created again")
	// } else {
	// 	if _, err = clientset.AppsV1beta1().Deployments("default").Update(deployment); err != nil {
	// 		return err
	// 	}
	// 	log.Println("found, updated")
	}
	return nil
}

func DeleteDeployment(clientset *kubernetes.Clientset, deployment *apps_v1beta1.Deployment) {
	var (
		sec int64
		prop metav1.DeletionPropagation
	)
	sec = 0
	prop = "Foreground"
	clientset.AppsV1beta1().Deployments("default").Delete(deployment.Name, &metav1.DeleteOptions{
		GracePeriodSeconds: &sec,
		PropagationPolicy: &prop,
	})
	log.Println("debug: deployment deleted:", deployment.Name)
}

func ApplyPod(clientset *kubernetes.Clientset, pod *v1.Pod) error {
	var err error
	if _, err = clientset.CoreV1().Pods("default").Get(pod.Name, metav1.GetOptions{}); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		// create one if not exist
		if _, err = clientset.CoreV1().Pods("default").Create(pod); err != nil {
			return err
		}
		log.Println("debug: pod not found, created")
	} else {	 // delete, then create
		if err = clientset.CoreV1().Pods("default").Delete(pod.Name, &metav1.DeleteOptions{}); err != nil {
			log.Println("warning: delete pod error:", err)
			return err
		} else {
			time.Sleep(2 * time.Second)
			if _, err = clientset.CoreV1().Pods("default").Create(pod); err != nil {
				return err
			}
		}
		log.Println("debug: pod found, deleted and created again")
	}
	return nil
}

func DeletePod(clientset *kubernetes.Clientset, pod *v1.Pod) {
	var (
		sec int64
		prop metav1.DeletionPropagation
	)
	sec = 0
	prop = "Foreground"
	clientset.CoreV1().Pods("default").Delete(pod.Name, &metav1.DeleteOptions{
		GracePeriodSeconds: &sec,
		PropagationPolicy: &prop,
	})
	log.Println("debug: pod deleted issued:", pod.Name)
}

func SetPodListenPort(pod *v1.Pod, port int) {
	containers := pod.Spec.Containers
	for j := range containers {
		if containers[j].Name == ClientContainerName {
			for k := range containers[j].Ports {
				if containers[j].Ports[k].Name == ClientListenPortName {
					containers[j].Ports[k].ContainerPort = (int32)(port)
					containers[j].Ports[k].HostPort      = (int32)(port)
					break
				}
			}
			for k := range containers[j].Env {
				if containers[j].Env[k].Name == ClientListenPortEnv {
					containers[j].Env[k].Value = strconv.Itoa(port)
					break
				}
			}
		}
	}
}

func GetPodListenPort(pod *v1.Pod) int {
	containers := pod.Spec.Containers
	for j := range containers {
		if containers[j].Name == ClientContainerName {
			for k := range containers[j].Ports {
				if containers[j].Ports[k].Name == ClientListenPortName {
					return int(containers[j].Ports[k].HostPort)
				}
			}
		}
	}
	return 0
}

func SetPodDeviceFraction(pod *v1.Pod, deviceFrac int) {
	containers := pod.Spec.Containers
	for j := range containers {
		if containers[j].Name == ClientContainerName {
			// containers[j].Resources.Limits[ResourceName] = deviceFrac
			// containers[j].Resources.Limits[ResourceName].Set(int64(deviceFrac))
			fracResource := resource.NewQuantity(int64(deviceFrac), resource.DecimalExponent)
			containers[j].Resources.Limits[ResourceName] = *fracResource
			break
		}
	}
}

func GetPodDeviceFraction(pod *v1.Pod) int {
	containers := pod.Spec.Containers
	for j := range containers {
		if containers[j].Name == ClientContainerName {
			deviceFrac, exist := containers[j].Resources.Limits[ResourceName]
			if exist {
				return int(deviceFrac.Value())
			}
		}
	}
	return 0
}

func IsConcernedPod(pod *v1.Pod) bool {
	return GetPodDeviceFraction(pod) > 0
	// return GetPodDeviceId(pod) >= 0
}