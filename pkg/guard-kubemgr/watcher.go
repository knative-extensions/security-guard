package guardkubemgr

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	spec "knative.dev/security-guard/pkg/apis/guard/v1alpha1"
)

func (k *KubeMgr) WatchOnce(ns string, set func(ns string, sid string, g *spec.GuardianSpec)) (e error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			e = fmt.Errorf("recovered from panic while watching cm and crd for ns %s! recover: %v", ns, recovered)
		}
	}()
	watcherCrd, err := k.crdClient.Guardians(ns).Watch(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("watch crd ns %s err %v", ns, err)
	}
	chCrd := watcherCrd.ResultChan()
	watcherCm, err := k.cmClient.CoreV1().ConfigMaps(ns).Watch(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("watch cm ns %s err %v", ns, err)
	}
	chCm := watcherCm.ResultChan()
	for {
		select {
		case event, ok := <-chCrd:
			if !ok {
				// the channel got closed, so we need to restart
				return fmt.Errorf("watch crd ns %s kubernetes hung up on us, restarting event watcher", ns)
			}

			// handle the crd event
			switch event.Type {
			case watch.Deleted:
				fallthrough
			case watch.Modified:
				fallthrough
			case watch.Added:
				g, ok := event.Object.(*spec.Guardian)
				if !ok {
					fmt.Printf("kubernetes cant convert to type Guardian\n")
					return
				}
				ns := g.ObjectMeta.Namespace
				sid := g.ObjectMeta.Name

				if event.Type == watch.Deleted {
					set(ns, sid, nil)
					continue
				}
				set(ns, sid, g.Spec)
			case watch.Error:
				s := event.Object.(*metav1.Status)
				fmt.Printf("Error during watch CRD: \n\tListMeta %v\n\tTypeMeta %v\n", s.ListMeta, s.TypeMeta)
			}
		case event, ok := <-chCm:
			if !ok {
				// the channel got closed, so we need to restart
				return fmt.Errorf("watch cm ns %s kubernetes hung up on us, restarting event watcher", ns)
			}

			// handle the cm event
			switch event.Type {
			case watch.Deleted:
				fallthrough
			case watch.Modified:
				fallthrough
			case watch.Added:
				cm, ok := event.Object.(*corev1.ConfigMap)
				if !ok {
					return fmt.Errorf("watch cm ns %s kubernetes cant convert to type configmap", ns)
				}
				ns := cm.ObjectMeta.Namespace
				sid := cm.ObjectMeta.Name
				if !strings.HasPrefix(sid, "guardian.") || strings.HasPrefix(sid, "guardian.ns.") {
					// skip...
					continue
				}
				if event.Type == watch.Deleted {
					set(ns, sid, nil)
					continue
				}
				if cm.Data["Guardian"] == "" {
					set(ns, sid, nil)
					continue
				}
				g := new(spec.GuardianSpec)
				gdata := []byte(cm.Data["Guardian"])
				jsonErr := json.Unmarshal(gdata, g)
				if jsonErr != nil {
					fmt.Printf("wsgate getConfig: unmarshel error %v", jsonErr)
					set(ns, sid, nil)
					continue
				}
				set(ns, sid, g)
			case watch.Error:
				s := event.Object.(*metav1.Status)
				fmt.Printf("Error during watch CM: \n\tListMeta %v\n\tTypeMeta %v\n", s.ListMeta, s.TypeMeta)
			}
		case <-time.After(10 * time.Minute):
			// deal with the issue where we get no events
			return fmt.Errorf("watch cm ns %s timeout, restarting event watcher", ns)
		}
	}
}
