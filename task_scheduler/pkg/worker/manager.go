package worker

import (

)

import (
	"fmt"
	"time"
	"log"
	"sync"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	clientgocache "k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/record"

	"task_scheduler/pkg/utils"
)

var (
	podToKey  = clientgocache.DeletionHandlingMetaNamespaceKeyFunc
	keyToInfo = clientgocache.SplitMetaNamespaceKey
)

type WorkerSetManager struct {
	clientset *kubernetes.Clientset

	// podLister can list/get pods from the shared informer's store.
	podLister corelisters.PodLister

	// // nodeLister can list/get nodes from the shared informer's store.
	// nodeLister corelisters.NodeLister
	
	// podInformerSynced returns true if the pod store has been synced at least once.
	podInformerSynced clientgocache.InformerSynced

	// // nodeInformerSynced returns true if the service store has been synced at least once.
	// nodeInformerSynced clientgocache.InformerSynced

	// podQueue is a rate limited work queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens. This
	// means we can ensure we only process a fixed amount of resources at a
	// time, and makes it easy to ensure we are never processing the same item
	// simultaneously in two different workers.
	podQueue workqueue.RateLimitingInterface

	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	recorder record.EventRecorder

	workerSet *WorkerSet

	// The cache to store the pod to be removed
	zombiePods map[string]*v1.Pod
	mut      *sync.Mutex
}

func NewWorkerSetManager(clientset *kubernetes.Clientset, stopCh <-chan struct{}) (*WorkerSetManager, error) {
	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(clientset, utils.ResyncPeriod)
	log.Printf("info: Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	// eventBroadcaster.StartLogging(log.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: clientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, v1.EventSource{Component: "schd-extender"})

	m := &WorkerSetManager{
		clientset:      clientset,
		podQueue:       workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "podQueue"),
		recorder:       recorder,
		zombiePods:     map[string]*v1.Pod{},
		mut:            new(sync.Mutex),
	}

	// Create pod informer.
	podInformer := kubeInformerFactory.Core().V1().Pods()
	podInformer.Informer().AddEventHandler(clientgocache.FilteringResourceEventHandler{
		FilterFunc: func(obj interface{}) bool {
			switch t := obj.(type) {
			case *v1.Pod:
				// log.Printf("debug: try to add pod %s in ns %s", t.Name, t.Namespace)
				// return utils.IsGPUsharingPod(t)
				return utils.IsConcernedPod(t)
			case clientgocache.DeletedFinalStateUnknown:
				if pod, ok := t.Obj.(*v1.Pod); ok {
					// log.Printf("debug: try to delete pod %s in ns %s", pod.Name, pod.Namespace)
					// return utils.IsGPUsharingPod(pod)
					return utils.IsConcernedPod(pod)
				}
				runtime.HandleError(fmt.Errorf("unable to convert object %T to *v1.Pod in %T", obj, m))
				return false
			default:
				runtime.HandleError(fmt.Errorf("unable to handle object in %T: %T", m, obj))
				return false
			}
		},
		Handler: clientgocache.ResourceEventHandlerFuncs{
			AddFunc:    m.addPodHandle,
			UpdateFunc: m.updatePodHandle,
			DeleteFunc: m.deletePodHandle,
		},
	})
	m.podLister = podInformer.Lister()
	m.podInformerSynced = podInformer.Informer().HasSynced

	// // Create node informer
	// nodeInformer := kubeInformerFactory.Core().V1().Nodes()
	// m.nodeLister = nodeInformer.Lister()
	// m.nodeInformerSynced = nodeInformer.Informer().HasSynced

	// Start informer goroutines.
	go kubeInformerFactory.Start(stopCh)

	// sync the clientgocache
	log.Println("info: begin to wait for cache")
	// if ok := clientgocache.WaitForCacheSync(stopCh, m.nodeInformerSynced); !ok {
	// 	return nil, fmt.Errorf("failed to wait for node caches to sync")
	// } else {
	// 	log.Println("info: init the node cache successfully")
	// }

	if ok := clientgocache.WaitForCacheSync(stopCh, m.podInformerSynced); !ok {
		return nil, fmt.Errorf("failed to wait for pod caches to sync")
	} else {
		log.Println("info: init the pod cache successfully")
	}
	log.Println("info: end to wait for cache")

	var err error
	// if m.clusterCache, err = newClusterCache(m.nodeLister, m.podLister, m.clientset); err != nil {
	// 	return nil, err
	// }
	if m.workerSet, err = newWorkerSet(m.clientset); err != nil {
		return nil, err
	}
	go m.Run(stopCh)
	return m, nil
	
}

func (m *WorkerSetManager) WorkerSet() *WorkerSet {
	return m.workerSet
}

func (m *WorkerSetManager) Run(stopCh <-chan struct{}) error {
	defer runtime.HandleCrash()
	defer m.podQueue.ShutDown()
	go wait.Until(m.runWorker, time.Second, stopCh)
	log.Println("info: Started runWorkers")
	<-stopCh
	log.Println("info: Shutting down runWorkers")
	return nil
}

func (m *WorkerSetManager) runWorker() {
	for m.processNextWorkItem() {

	}
}

func (m *WorkerSetManager) processNextWorkItem() bool {
	log.Print("trace: begin processNextWorkItem()")
	key, quit := m.podQueue.Get()
	if quit {
		return false
	}
	defer m.podQueue.Done(key)
	defer log.Print("trace: end processNextWorkItem()")
	success, err := m.syncPod(key.(string))
	if err == nil {
		// log.Printf("Error syncing pods: %v", err)
		if success {
			m.podQueue.Forget(key)
		}
		return false
	}

	log.Printf("error: Error syncing pods: %v", err)
	runtime.HandleError(fmt.Errorf("Error syncing pod: %v", err))
	m.podQueue.AddRateLimited(key)

	return true
}

// syncPod will sync the pod with the given key if it has had its expectations fulfilled,
// meaning it did not expect to see any more of its pods created or deleted. 
// This function is not meant to be invoked concurrently with the same key.
func (m *WorkerSetManager) syncPod(key string) (success bool, err error) {
	ns, name, err := keyToInfo(key)
	log.Printf("trace: begin to sync pod %s in ns %s", name, ns)
	if err != nil {
		return false, err
	}

	pod, err := m.podLister.Pods(ns).Get(name)
	switch {
	case errors.IsNotFound(err):
		log.Printf("trace: pod %s in ns %s has been deleted.", name, ns)
		m.mut.Lock()
		pod, found := m.zombiePods[key]
		if found {
			// m.clusterCache.removePod(pod)
			m.workerSet.removeWorkerPod(pod)
			delete(m.zombiePods, key)
		}
		m.mut.Unlock()
	case err != nil:
		log.Printf("warn: unable to retrieve pod %v from the store: %v", key, err)
	default:
		// if err := m.clusterCache.addOrUpdatePod(pod); err != nil {
		// 	return false, err
		// }
		if err := m.workerSet.updateWorkerPod(pod); err != nil {
			return false, err
		}
		// if utils.IsCompletePod(pod) {
		// 	log.Printf("debug: pod %s in ns %s has completed.", name, ns)
		// 	c.schedulerCache.RemovePod(pod)
		// } else {
		// 	err := c.schedulerCache.AddOrUpdatePod(pod)
		// 	if err != nil {
		// 		return false, err
		// 	}
		// }
	}

	return true, nil
}

func (m *WorkerSetManager) addPodHandle(obj interface{}) {
	pod, ok := obj.(*v1.Pod)
	if !ok {
		log.Printf("warn: cannot convert to *v1.Pod: %v", obj)
		return
	}

	podKey, err := podToKey(pod)
	if err != nil {
		log.Printf("warn: Failed to get the jobkey: %v", err)
		return
	}
	
	log.Printf("trace: add pod %s in ns %s to cache, issued to podQueue", pod.Name, pod.Namespace)
	m.podQueue.Add(podKey)
}

func (m *WorkerSetManager) updatePodHandle(oldObj, newObj interface{}) {
	oldPod, ok := oldObj.(*v1.Pod)
	if !ok {
		log.Printf("warn: cannot convert oldObj to *v1.Pod: %v", oldObj)
		return
	}
	newPod, ok := newObj.(*v1.Pod)
	if !ok {
		log.Printf("warn: cannot convert newObj to *v1.Pod: %v", newObj)
		return
	}
	// log.Printf("debug: update pod %s in ns %s", newPod.Name, newPod.Namespace)
	if oldPod.Status.Phase == newPod.Status.Phase {
		log.Printf("trace: refuse to update pod name %s in ns %s (old status %v, new status %v)", 
			newPod.Name,
			newPod.Namespace,
			oldPod.Status.Phase,
			newPod.Status.Phase)
		return
	}
	log.Printf("trace: Need to update pod name %s in ns %s (old status %v, new status %v); its old annotation %v, new annotation %v",
		newPod.Name,
		newPod.Namespace,
		oldPod.Status.Phase,
		newPod.Status.Phase,
		oldPod.Annotations,
		newPod.Annotations)
	podKey, err := podToKey(newPod)
	if err != nil {
		log.Printf("warn: Failed to get the jobkey: %v", err)
		return
	}
	m.podQueue.Add(podKey)
	return
	
	// needUpdate := false
	// podUID := oldPod.UID
	// // 1. Need update when pod is turned to complete or failed
	// if c.schedulerCache.KnownPod(podUID) && utils.IsCompletePod(newPod) {
	// 	needUpdate = true
	// }
	// // 2. Need update when it's unknown pod, and GPU annotation has been set
	// if !c.schedulerCache.KnownPod(podUID) && utils.GetGPUIDFromAnnotation(newPod) >= 0 {
	// 	needUpdate = true
	// }
	// if needUpdate {
	// 	podKey, err := KeyFunc(newPod)
	// 	if err != nil {
	// 		log.Printf("warn: Failed to get the jobkey: %v", err)
	// 		return
	// 	}
	// 	log.Printf("info: Need to update pod name %s in ns %s and old status is %v, new status is %v; its old annotation %v and new annotation %v",
	// 		newPod.Name,
	// 		newPod.Namespace,
	// 		oldPod.Status.Phase,
	// 		newPod.Status.Phase,
	// 		oldPod.Annotations,
	// 		newPod.Annotations)
	// 	c.podQueue.Add(podKey)
	// } else {
	// 	log.Printf("debug: No need to update pod name %s in ns %s and old status is %v, new status is %v; its old annotation %v and new annotation %v",
	// 		newPod.Name,
	// 		newPod.Namespace,
	// 		oldPod.Status.Phase,
	// 		newPod.Status.Phase,
	// 		oldPod.Annotations,
	// 		newPod.Annotations)
	// }
}

func (m *WorkerSetManager) deletePodHandle(obj interface{}) {
	var pod *v1.Pod
	switch t := obj.(type) {
	case *v1.Pod:
		pod = t
	case clientgocache.DeletedFinalStateUnknown:
		var ok bool
		pod, ok = t.Obj.(*v1.Pod)
		if !ok {
			log.Printf("warn: cannot convert to *v1.Pod: %v", t.Obj)
			return
		}
	default:
		log.Printf("warn: cannot convert to *v1.Pod: %v", t)
		return
	}

	log.Printf("trace: delete pod %s in ns %s", pod.Name, pod.Namespace)
	podKey, err := podToKey(pod)
	if err != nil {
		log.Printf("warn: Failed to get the jobkey: %v", err)
		return
	}
	m.podQueue.Add(podKey)
	m.mut.Lock()
	m.zombiePods[podKey] = pod
	m.mut.Unlock()
}
