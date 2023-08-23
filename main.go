package main

// 实现一个小的功能
// 在部署一个特定资源的时候 把日志收集给搞上

import (
	"context"
	v1 "k8s.io/api/apps/v1"
	v13 "k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// 第一步
	// 获取config
	// 可以有两种方式 一种使用外部config文件 一种是在在集群中调用
	config, err := rest.InClusterConfig()
	if err != nil {
		klog.Fatal(err)
	}
	// 第二步
	// 创建一个新的clientset对象
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		klog.Fatal(err)
	}
	// 第三步
	// 构建一个新的informer factory对象
	factory := informers.NewSharedInformerFactory(clientset, 0)
	// 说明资源对象的informer
	informer := factory.Apps().V1().Deployments().Informer()
	// 给informer添加handler 需要实现 OnAdd OnUpdate OnDelete 三个方法
	_, err = informer.AddEventHandler(&informerHandler{
		clientset: clientset,
	})
	if err != nil {
		return
	}
	stop := make(chan struct{}, 2)
	// 调用RUN函数 需要传入停止channel
	go informer.Run(stop)
	forever := make(chan os.Signal, 1)
	signal.Notify(forever, syscall.SIGINT, syscall.SIGTERM)
	<-forever
	stop <- struct{}{}
	close(forever)
	close(stop)
}

type informerHandler struct {
	clientset *kubernetes.Clientset
}

func (i *informerHandler) OnUpdate(oldObj, newObj interface{}) {

}

func (i *informerHandler) OnDelete(obj interface{}) {

}

func (i *informerHandler) OnAdd(obj interface{}) {
	dp := obj.(*v1.Deployment)
	if dp.ObjectMeta.Annotations["needFluentd"] == "yes" {

		dp2, err := i.clientset.AppsV1().Deployments(dp.Namespace).Get(context.TODO(), dp.Name, v12.GetOptions{})
		klog.Infof("ADD: the old version %s %s", dp2.Name, dp2.ObjectMeta.ResourceVersion)
		fluentContainer := v13.Container{Name: "fluentd-sidecar",
			Image: "fluent/fluentd:v1.15-debian-1",
			Env: []v13.EnvVar{
				v13.EnvVar{
					Name:  "FLUENTD_CONF",
					Value: "fluentd.conf",
				},
			},
			VolumeMounts: []v13.VolumeMount{
				v13.VolumeMount{
					Name:      "config-volume",
					ReadOnly:  false,
					MountPath: "/fluentd/etc",
				},
			}}
		fluentVolumne := v13.Volume{
			Name: "config-volume",
			VolumeSource: v13.VolumeSource{
				ConfigMap: &v13.ConfigMapVolumeSource{
					LocalObjectReference: v13.LocalObjectReference{Name: "fluentd-config-sidecar"},
				},
			},
		}
		dp2.Spec.Template.Spec.Containers = append(dp2.Spec.Template.Spec.Containers, fluentContainer)
		dp2.Spec.Template.Spec.Volumes = append(dp2.Spec.Template.Spec.Volumes, fluentVolumne)
		dp2, err = i.clientset.AppsV1().Deployments(dp2.Namespace).Update(context.Background(), dp2, v12.UpdateOptions{})
		if err != nil {
			klog.Infoln(err)
		}
		dp = dp2.DeepCopy()
		klog.Infof("ADD: the new version %s %s", dp2.Name, dp2.ObjectMeta.ResourceVersion)
		return
	}

	// resourceVersion should not be set on objects to be created
	// 可能的处理方法 需要先更新这个deploy
	if dp.Status.AvailableReplicas >= *dp.Spec.Replicas {
		return
	}
	_, err := i.clientset.AppsV1().Deployments(dp.Namespace).Create(context.Background(), dp, v12.CreateOptions{})
	if err != nil {

		klog.Infoln(err)
	}
}
