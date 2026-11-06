# 真果鉴开发约定

## 项目目标

构建 Flutter 多端独立应用。站源请求、解析、在线播放及后续下载、合并均在设备端完成；成品不依赖旧短剧库程序、旧项目路径或远程自建服务。可以迁移已经验证的业务逻辑，并将依赖一并纳入本项目。

## 优先级与交付

1. Android 手机和 Windows 电脑同为第一优先级。
2. Android TV 第二优先级，适配遥控焦点与电视布局。
3. iOS 最后实施，单独处理原生媒体库和后台任务限制。
4. 首版优先完成可用的在线播放闭环：浏览、搜索、剧集详情、选集、播放、错误重试。
5. 其他功能按 README.md 中的待办顺序逐项实现，完成后更新状态。不要为了未到优先级的功能推迟首版。

## 实施与验证

按用户 2026-09-21 的要求，目前完成功能优先，暂停测试与回归，待用户安排后集中执行。源码格式整理及任务收尾同步仍执行；未验证的实现明确标为开发快照，不作为通过验收的版本。

恢复集中验证后，先完整实现当前交付范围，再集中执行静态检查、构建、自动化和设备验证；不要每改一个小功能就测试。集中验证发现问题后，修复并执行相关复验。未验证的平台和行为必须明确标注，不得将生成源码等同于可用成品。

直接在本项目中维护源码。临时目录仅放工具、构建缓存和验证产物，不在临时目录维护另一份正式实现。不要修改旧短剧库和果果剧库。

## 任务收尾同步

每次完成开发、修复或文档修改后，在必要检查通过、最终回复之前，必须执行 `python3 scripts/finish_task.py --message "本次实际完成的变更"`。脚本只将干净源码同步到本项目同级的 `../guoapp`，并生成 `真果·鉴-YYYYMMDDHHMM.zip` 纯源码压缩包；不要求目标存在 `.git`，不执行提交、tag、分支、推送或其他 Git 操作。同一分钟内重复执行会追加序号，不覆盖已有压缩包，作为可恢复的版本。有后续修改时重新执行收尾，并用 `python3 scripts/sync_source.py --check` 确认一致；失败先处理原因，不要声称已完成。

`guoapp` 是面向 GitHub 的纯源码镜像，不依赖本地 Git 仓库；其余文件与导出源码保持一致，包含源码、必要资源、构建配置、锁文件、测试和 Actions，排除 SDK、第三方依赖目录、缓存、编译产物、签名文件和个人配置。目标目录若已有 `.git`，同步时原样保留但不读取、不修改、不删除。新增源码根目录或构建输入时同步维护脚本的收录规则。

功能版本递增 `pubspec.yaml` 的版本号和构建号。源码恢复以版本号、源码压缩包和 README 记录为依据；未通过检查的代码不得标为已完成版本。

如果正在 `guoapp` 本身工作，不向自身同步；脚本会跳过此情况。不要为了同步另建临时源码副本。

## 代码与资料

保持单一 README.md 使用说明，并在其中维护优先级待办与平台验证状态；不要创建零散 docs 文档。AGENTS.md 仅记录开发约定。

不新增解释性代码注释，必要的编译或工具指令除外。采用清晰命名和模块边界。不要提交密钥、个人配置、下载的视频、SDK 或构建缓存。

不要查看、下载、分析或处理站源图片。页面可正常显示海报地址；开发验证使用合成数据或拦截图片请求。用户直接提供的界面截图可以查看。

遵守先实现后集中验证的节奏；未经用户明确要求，不启动子 AGENT。

## 代码结构与构建速查

- `lib/` 保持扁平：页面、播放器、下载、本地媒体库与 FFI 封装（`core_bridge.dart`）；新文件按功能用 snake_case 命名，不建子目录。
- `native/core` 是 Go 站源核心（`provider_*.go` 承载各站源协议，另有缓存、下载、弹幕、局域网逻辑），`native/bridge/main.go` 是 C ABI 入口；站源解析在 Go 侧实现并配套 `*_test.go`，Dart 经 FFI 调用，不重复实现协议。Go 模块名为 `duanjuapp/native`。
- 播放使用 media_kit / libmpv；FFmpegKit 仅用于合并与导出，不参与播放和下载转码。`packages/media_kit_libs_windows_video` 经 pubspec `dependency_overrides` 本地固定，Windows 媒体依赖不随意升级。
- 红果取流首选 `github.com/beilunyang/openhongguo`（Apache-2.0，匿名设备 + 官方签名，可返回全部分集码流与 CENC 密钥），匿名档案保存在应用数据目录；旧 App 接口、网页公开文件与备用解析服务保留为回退，顺序见 `native/core/provider_hongguo.go` 的 `resolveHongguoMedia`。升级该依赖后需重跑 `CHECK_LIVE_VIP=true go test -run TestLiveHongguoOpenPlayback ./core`（需要时自行补回该用例）。
- 两版构建：默认红果鉴（仅红果站源）与全站源真果鉴。三处开关必须一致：构建脚本 `--all-sources`、Flutter `--dart-define=ALL_SOURCES=true`、Go ldflag `buildAllSources=true`；应用启动时校验，避免原生库混装。
- 全站源版只开放实测可播放的站源：白名单在 `native/core/app_build.go` 的 `playableProviderSources` 与 `lib/models.dart` 的 `knownValues`，两处必须同步；新增或恢复站源先跑 `CHECK_LIVE_ALL=true` 复测，通过后再加入白名单。其余站源解析代码保留，不删除。
- 构建入口：首次调试先 `python3 scripts/build_native.py --platform android --abi arm64-v8a`（Windows 用 `--platform windows`），再 `python3 scripts/build_android.py`、`.\scripts\build_windows.ps1`、`python3 scripts/build_ios.py`，均可加 `--all-sources` 和国内镜像开关；产物在 `dist/`。工具链：Flutter 3.47.4、Dart 3.12+、Go 1.24.1+、JDK 17、SDK 36、NDK 28.2.13676358。完整命令见 README「开发与构建」。
- 集中验证命令（当前按上文暂停执行，恢复后运行）：`python3 -m unittest discover -s scripts -p 'test_*.py'`；`dart format --output=none --set-exit-if-changed lib test integration_test test_driver`；`dart analyze --fatal-infos lib test integration_test test_driver`；`flutter test --dart-define=DISABLE_REMOTE_IMAGES=true`（全站源版另加 `--dart-define=ALL_SOURCES=true`）；`cd native && go test -race ./...`（全站源版另加 `-ldflags="-X duanjuapp/native/core.buildAllSources=true"`）。
- 开发与测试全程使用 `--dart-define=DISABLE_REMOTE_IMAGES=true`，不请求站源图片，与上文图片约定一致。
- 本目录不是 Git 仓库，不执行任何 Git 操作；版本快照由 `scripts/finish_task.py` 生成的源码压缩包承担，详见「任务收尾同步」。
