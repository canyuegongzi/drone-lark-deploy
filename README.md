# Drone Lark Plugin 飞书通知插件

## Sample

```yml
- name: deploy-notice
  depends_on: [deploy]
  image: registry.cn-hangzhou.aliyuncs.com/canyuegongzi/drone-plugin-feishu:4.0
  pull: if-not-exists
  when:
    status:
      - success
      - failure
  settings:
    messagetype: DEPLOY
    dockergroup:
      from_secret: docker_image_group
    webhook:
      from_secret: feishu_notice_webhook
    secret:
      from_secret: feishu_notice_secret
    debug: true

```

## Build

```bash
vi ./build.sh
# replace ydq1234 => your account or docker hub domain

chmod +x ./build.sh

#export tag=xxx  （tag default is latest）
./build.sh
```