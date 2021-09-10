# wordpress-operator

### 前置条件

* 安装 Docker Desktop，并启动内置的 Kubernetes 集群
* 注册一个 [hub.docker.com](https://hub.docker.com/) 账户，需要将本地构建好的镜像推送至公开仓库中
* 安装 operator SDK CLI: `brew install operator-sdk`
* 安装 Go: `brew install go`

本示例推荐的依赖版本：

* Docker Desktop: >= 4.0.0
* Kubernetes: >= 1.21.4
* Operator-SDK: >= 1.11.0
* Go: >= 1.17

### 创建项目

使用 Operator SDK CLI 创建名为 wordpress-operator 的项目。

```shell
mkdir -p $HOME/projects/wordpress-operator
cd $HOME/projects/wordpress-operator
go env -w GOPROXY=https://goproxy.cn,direct

operator-sdk init \
--domain=jxlwqq.github.io \
--repo=github.com/jxlwqq/wordpress-operator \
--skip-go-version-check
```

### 创建 API 和控制器

使用 Operator SDK CLI 创建自定义资源定义（CRD）API 和控制器。

运行以下命令创建带有组 app、版本 v1alpha1 和种类 Wordpress 的 API：

```shell
operator-sdk create api \
--resource=true \
--controller=true \
--group=app \
--version=v1alpha1 \
--kind=Wordpress
```

定义 Wordpress 自定义资源（CR）的 API。

修改 api/v1alpha1/wordpress_types.go 中的 Go 类型定义，使其具有以下 spec 和 status

```go
type WordpressSpec struct {
	Size int32 `json:"size"`
	Version string `json:"version"`
}
```


为资源类型更新生成的代码：
```shell
make generate
```

运行以下命令以生成和更新 CRD 清单：
```shell
make manifests
```

#### 实现控制器

> 由于逻辑较为复杂，代码较为庞大，所以无法在此全部展示，完整的操作器代码请参见 controllers 目录。
在本例中，将生成的控制器文件 controllers/wordpress_controller.go 替换为以下示例实现：

```go
/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"time"

	appv1alpha1 "github.com/jxlwqq/wordpress-operator/api/v1alpha1"
)

// WordpressReconciler reconciles a Wordpress object
type WordpressReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=app.jxlwqq.github.io,resources=wordpresses,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=app.jxlwqq.github.io,resources=wordpresses/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=app.jxlwqq.github.io,resources=wordpresses/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Wordpress object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.9.2/pkg/reconcile
func (r *WordpressReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := ctrllog.FromContext(ctx)
	reqLogger.Info("---Reconciling Wordpress---")

	wordpress := &appv1alpha1.Wordpress{}
	err := r.Client.Get(ctx, req.NamespacedName, wordpress)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	var result *reconcile.Result

	// MySQL
	reqLogger.Info("---MySQL Secret---")
	result, err = r.ensureSecret(r.secretForMysql(wordpress))
	if result != nil {
		return *result, err
	}

	reqLogger.Info("---MySQL PVC---")
	result, err = r.ensurePVC(r.pvcForMysql(wordpress))
	if result != nil {
		return *result, err
	}

	reqLogger.Info("---MySQL Deployment---")
	result, err = r.ensureDeployment(r.deploymentForMysql(wordpress))
	if result != nil {
		return *result, err
	}

	reqLogger.Info("---MySQL Service---")
	result, err = r.ensureService(r.serviceForMysql(wordpress))
	if result != nil {
		return *result, err
	}

	reqLogger.Info("---MySQL Check Status---")
	if !r.isMysqlUp(wordpress) {
		delay := time.Second * time.Duration(5)
		return ctrl.Result{RequeueAfter: delay}, nil
	}

	// WordPress
	reqLogger.Info("---WordPress PVC---")
	result, err = r.ensurePVC(r.pvcForWordpress(wordpress))
	if result != nil {
		return *result, err
	}

	reqLogger.Info("---WordPress Deployment---")
	result, err = r.ensureDeployment(r.deploymentForWordpress(wordpress))
	if result != nil {
		return *result, err
	}

	reqLogger.Info("---WordPress Service---")
	result, err = r.ensureService(r.serviceForWordpress(wordpress))
	if result != nil {
		return *result, err
	}

	reqLogger.Info("---WordPress Handle Changes---")
	result, err = r.handleWordpressChanges(wordpress)
	if result != nil {
		return *result, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *WordpressReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appv1alpha1.Wordpress{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		Owns(&corev1.Secret{}).
		Owns(&corev1.Service{}).
		Complete(r)
}
```


运行以下命令以生成和更新 CRD 清单：
```shell
make manifests
```

#### 运行 Operator

捆绑 Operator，并使用 Operator Lifecycle Manager（OLM）在集群中部署。

修改 Makefile 中 IMAGE_TAG_BASE 和 IMG：

```makefile
IMAGE_TAG_BASE ?= docker.io/jxlwqq/wordpress-operator
IMG ?= $(IMAGE_TAG_BASE):latest
```

构建镜像：

```shell
make docker-build
```

将镜像推送到镜像仓库：
```shell
make docker-push
```

成功后访问：https://hub.docker.com/r/jxlwqq/wordpress-operator


运行 make bundle 命令创建 Operator 捆绑包清单，并依次填入名称、作者等必要信息:
```shell
make bundle
```

构建捆绑包镜像：
```shell
make bundle-build
```

推送捆绑包镜像：
```shell
make bundle-push
```

成功后访问：https://hub.docker.com/r/jxlwqq/wordpress-operator-bundle