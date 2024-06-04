# 为 seata-go 做贡献

如果你有兴趣为 seata-go 贡献代码，我们会热烈欢迎。首先，我们非常鼓励这种意愿。这是为您提供的贡献指南列表。

[[English Contributing Document](./CONTRIBUTING.md)]

## 话题

* [报告一般问题](#报告一般问题)
* [代码和文档贡献](#代码和文档贡献)
* [测试用例贡献](#测试用例贡献)
* [参与帮助任何事情](#参与帮助任何事情)
* [代码风格](#代码风格)

## 报告一般问题

老实说我们把每一个 seata-go 用户都视为非常善良的贡献者。在体验了 seata-go 之后，您可能会对项目有一些反馈。然后随时通过 [NEW ISSUE](https://github.com/apache/incubator-seata-go/issues/new/choose)打开一个问题。

因为我们在一个分布式的方式合作项目 seata-go，我们欣赏写得很好的，详细的，准确的问题报告。为了让沟通更高效，我们希望每个人都可以搜索您的问题是否在搜索列表中。如果您发现它存在，请在现有问题下的评论中添加您的详细信息，而不是打开一个全新的问题。

为了使问题细节尽可能标准，我们为问题报告者设置了一个[问题模板](./.github/ISSUE_TEMPLATE) 请务必按照说明填写模板中的字段。

有很多情况你可以打开一个问题：

* 错误报告
* 功能要求
* 性能问题
* 功能提案
* 功能设计
* 需要帮助
* 文档不完整
* 测试改进
* 关于项目的任何问题
* 等等

另外我们必须提醒的是，在填写新问题时，请记住从您的帖子中删除敏感数据。敏感数据可能是密码、密钥、网络位置、私人业务数据等。

## 代码和文档贡献

鼓励采取一切措施使 seata-go 项目变得更好。在 GitHub 上，seata-go 的每项改进都可以通过 PR（Pull Request 的缩写）实现。

* 如果您发现错别字，请尝试修复它！
* 如果您发现错误，请尝试修复它！
* 如果您发现一些多余的代码，请尝试删除它们！
* 如果您发现缺少一些测试用例，请尝试添加它们！
* 如果您可以增强功能，请**不要**犹豫！
* 如果您发现代码晦涩难懂，请尝试添加注释以使其更加易读！
* 如果您发现代码丑陋，请尝试重构它！
* 如果您能帮助改进文档，那就再好不过了！
* 如果您发现文档不正确，只需执行并修复它！
* ...

实际上不可能完整地列出它们。记住一个原则：

> 我们期待您的任何PR。

由于您已准备好通过 PR 改进 seata-go，我们建议您可以在此处查看 PR 规则。

* [工作区准备](#工作区准备)
* [分支定义](#分支定义)
* [提交规则](#提交规则)
* [PR说明](#PR说明)

### 工作区准备

为了提出 PR，我们假设你已经注册了一个 GitHub ID。然后您可以通过以下步骤完成准备工作：

1. **FORK** seata-go 到您的存储库。要完成这项工作，您只需单击 [apache/seata-go](https://github.com/apache/incubator-seata-go) 主页右侧的 Fork 按钮。然后你将在 中得到你的存储库`https://github.com/<your-username>/seata-go`，其中 your-username 是你的 GitHub 用户名。

2. **克隆** 您自己的存储库以在本地开发. 用于 `git clone git@github.com:<your-username>/seata-go.git` 将存储库克隆到本地计算机。 然后您可以创建新分支来完成您希望进行的更改。

3. **设置远程** 将上游设置为 `git@github.com:apache/seata-go.git` 使用以下两个命令：

```bash
git remote add upstream git@github.com:apache/seata-go.git
git remote set-url --push upstream no-pushing
```

使用此远程设置，您可以像这样检查您的 git 远程配置：

```shell
$ git remote -v
origin     git@github.com:<your-username>/seata-go.git (fetch)
origin     git@github.com:<your-username>/seata-go.git (push)
upstream   git@github.com:apache/seata-go.git (fetch)
upstream   no-pushing (push)
```

添加这个，我们可以轻松地将本地分支与上游分支同步。

### 分支定义

现在我们假设通过拉取请求的每个贡献都是针对 seata-go 中的 [开发分支](https://github.com/apache/incubator-seata-go/tree/master) 。在贡献之前，请注意分支定义会很有帮助。

作为贡献者，请再次记住，通过拉取请求的每个贡献都是针对分支开发的。而在 seata-go 项目中，还有其他几个分支，我们一般称它们为 release 分支（如0.6.0、0.6.1）、feature 分支、hotfix 分支和 master 分支。

当正式发布一个版本时，会有一个发布分支并以版本号命名。

在发布之后，我们会将发布分支的提交合并到主分支中。

当我们发现某个版本有 bug 时，我们会决定在以后的版本中修复它，或者在特定的 hotfix 版本中修复它。当我们决定修复 hotfix 版本时，我们会根据对应的 release 分支 checkout hotfix 分支，进行代码修复和验证，合并到 develop 分支和 master 分支。

对于较大的功能，我们将拉出功能分支进行开发和验证。


### 提交规则

实际上，在 seata-go 中，我们在提交时会认真对待两条规则：

* [提交消息](#提交消息)
* [提交内容](#提交内容)

#### 提交消息

提交消息可以帮助审稿人更好地理解提交 PR 的目的是什么。它还可以帮助加快代码审查过程。我们鼓励贡献者使用显式的提交信息，而不是模糊的信息。一般来说，我们提倡以下提交消息类型：

* docs: xxxx. For example, "docs: add docs about seata-go cluster installation".
* feature: xxxx.For example, "feature: support oracle in AT mode".
* bugfix: xxxx. For example, "bugfix: fix panic when input nil parameter".
* optimize: xxxx. For example, "optimize: simplify to make codes more readable".
* test: xxx. For example, "test: add unit test case for func InsertIntoArray".
* 其他可读和显式的表达方式。

另一方面，我们不鼓励贡献者通过以下方式提交消息：

* ~~修复错误~~
* ~~更新~~
* ~~添加文档~~

如果你不知道该怎么做，请参阅 [如何编写 Git 提交消息](http://chris.beams.io/posts/git-commit/) 作为开始。

#### 提交内容

提交内容表示一次提交中包含的所有内容更改。我们最好在一次提交中包含可以支持审阅者完整审查的内容，而无需任何其他提交的帮助。换句话说，一次提交中的内容可以通过 CI 以避免代码混乱。简而言之，我们需要牢记三个小规则：

* 避免在提交中进行非常大的更改；
* 每次提交都完整且可审查。
* 提交时检查 git config(`user.name`, `user.email`) 以确保它与您的 GitHub ID 相关联。

```bash
git config --get user.name
git config --get user.email
```

* 提交pr时，请在'changes/'文件夹下的dev.md文件中添加当前更改的简要说明


另外，在代码变更部分，我们建议所有贡献者阅读 seata-go 的 [代码风格](#代码风格)。

无论是提交信息，还是提交内容，我们都更加重视代码审查。

#### 格式化代码

提交代码之前，在项目根目录下执行格式化代码的脚本：

~~~shell
sh goimports.sh
~~~


### PR说明

PR 是更改 seata-go 项目文件的唯一方法。为了帮助审查人更好地理解你的目的，PR 描述不能太详细。我们鼓励贡献者遵循 [PR 模板](./.github/PULL_REQUEST_TEMPLATE.md) 来完成拉取请求。

## 测试用例贡献

任何测试用例都会受到欢迎。目前，seata-go 功能测试用例是高优先级的。

* 对于单元测试，在文件目录下创建一个 xxx_test.go 文件即可

## 参与帮助任何事情

我们选择 GitHub 作为 seata-go 协作的主要场所。所以 seata-go 的最新更新总是在这里。尽管通过 PR 贡献是一种明确的帮助方式，但我们仍然呼吁其他方式。

* 如果可以的话，回复别人的问题；
* 帮助解决其他用户的问题；
* 帮助审查他人的 PR 设计；
* 帮助审查其他人在 PR 中的代码；
* 讨论 seata-go以使事情更清楚；
* 在Github之外宣传 seata-go 技术；
* 写关于 seata-go 的博客等等。


## 代码风格

Seata-go 代码风格参考[uber-go/guide](https://github.com/uber-go/guide) 。


### IDE插件安装（非必须）

*没有必要安装，如果你想在编码的时候发现问题。*

安装 go fmt 和 goimports 插件，详情参考：https://github.com/golang/tools
