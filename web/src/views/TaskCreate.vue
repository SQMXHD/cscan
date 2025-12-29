<template>
  <div class="task-create-page">
    <el-card class="create-card">
      <el-form ref="formRef" :model="form" :rules="rules" label-width="120px" class="task-form">
        <!-- 基本信息 -->
        <el-form-item label="任务名称" prop="name">
          <el-input v-model="form.name" placeholder="请输入任务名称" />
        </el-form-item>
        <el-form-item label="扫描目标" prop="target">
          <el-input v-model="form.target" type="textarea" :rows="6" placeholder="每行一个目标，支持格式:&#10;• IP: 192.168.1.1&#10;• CIDR: 192.168.1.0/24&#10;• IP范围: 192.168.1.1-192.168.1.100&#10;• 域名: example.com" />
        </el-form-item>
        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item label="工作空间">
              <el-select v-model="form.workspaceId" placeholder="选择工作空间" clearable style="width: 100%">
                <el-option v-for="ws in workspaces" :key="ws.id" :label="ws.name" :value="ws.id" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="所属组织">
              <el-select v-model="form.orgId" placeholder="选择组织" clearable style="width: 100%">
                <el-option v-for="org in organizations" :key="org.id" :label="org.name" :value="org.id" />
              </el-select>
            </el-form-item>
          </el-col>
        </el-row>
        <el-form-item label="指定Worker">
          <el-select v-model="form.workers" multiple placeholder="不选则任意Worker执行" clearable style="width: 100%">
            <el-option v-for="w in workers" :key="w.name" :label="`${w.name} (${w.ip})`" :value="w.name" />
          </el-select>
        </el-form-item>
        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item label="定时任务">
              <el-switch v-model="form.isCron" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item v-if="form.isCron" label="Cron表达式">
              <el-input v-model="form.cronRule" placeholder="0 0 * * *" />
            </el-form-item>
          </el-col>
        </el-row>

        <!-- 可折叠配置区域 -->
        <el-collapse v-model="activeCollapse" class="config-collapse">
          <!-- 子域名扫描 -->
          <el-collapse-item name="domainscan">
            <template #title>
              <span class="collapse-title">子域名扫描 <el-tag v-if="form.domainscanEnable" type="success" size="small">开</el-tag></span>
            </template>
            <el-form-item label="启用">
              <el-switch v-model="form.domainscanEnable" />
              <span class="form-hint">针对域名目标进行子域名枚举</span>
            </el-form-item>
            <template v-if="form.domainscanEnable">
              <el-form-item label="使用Subfinder">
                <el-switch v-model="form.domainscanSubfinder" />
                <span class="form-hint">使用Subfinder进行子域名枚举</span>
              </el-form-item>
              <el-row :gutter="20">
                <el-col :span="12">
                  <el-form-item label="超时时间(秒)">
                    <el-input-number v-model="form.domainscanTimeout" :min="60" :max="3600" style="width:100%" />
                  </el-form-item>
                </el-col>
                <el-col :span="12">
                  <el-form-item label="最大枚举时间(分)">
                    <el-input-number v-model="form.domainscanMaxEnumTime" :min="1" :max="60" style="width:100%" />
                  </el-form-item>
                </el-col>
              </el-row>
              <el-row :gutter="20">
                <el-col :span="12">
                  <el-form-item label="速率限制">
                    <el-input-number v-model="form.domainscanRateLimit" :min="0" :max="1000" style="width:100%" />
                    <span class="form-hint">0=不限制</span>
                  </el-form-item>
                </el-col>
              </el-row>
              <el-form-item label="扫描选项">
                <el-checkbox v-model="form.domainscanRemoveWildcard">移除泛解析域名</el-checkbox>
              </el-form-item>
              <el-form-item label="DNS解析">
                <el-checkbox v-model="form.domainscanResolveDNS">解析子域名DNS</el-checkbox>
                <span class="form-hint">并发数由Worker设置控制</span>
              </el-form-item>
            </template>
          </el-collapse-item>

          <!-- 端口扫描 -->
          <el-collapse-item name="portscan">
            <template #title>
              <span class="collapse-title">端口扫描 <el-tag v-if="form.portscanEnable" type="success" size="small">开</el-tag></span>
            </template>
            <el-form-item label="启用">
              <el-switch v-model="form.portscanEnable" />
            </el-form-item>
            <template v-if="form.portscanEnable">
              <el-form-item label="扫描工具">
                <el-radio-group v-model="form.portscanTool">
                  <el-radio label="naabu">Naabu (推荐)</el-radio>
                  <el-radio label="masscan">Masscan</el-radio>
                </el-radio-group>
              </el-form-item>
              <el-form-item label="端口范围">
                <el-select v-model="form.ports" filterable allow-create default-first-option style="width: 100%">
                  <el-option label="top100 - 常用100端口" value="top100" />
                  <el-option label="top1000 - 常用1000端口" value="top1000" />
                  <el-option label="80,443,8080,8443 - Web常用" value="80,443,8080,8443" />
                  <el-option label="1-65535 - 全端口" value="1-65535" />
                </el-select>
              </el-form-item>
              <el-row :gutter="20">
                <el-col :span="12">
                  <el-form-item label="扫描速率">
                    <el-input-number v-model="form.portscanRate" :min="100" :max="100000" style="width:100%" />
                  </el-form-item>
                </el-col>
                <el-col :span="12">
                  <el-form-item label="端口阈值">
                    <el-input-number v-model="form.portThreshold" :min="0" :max="65535" style="width:100%" />
                  </el-form-item>
                </el-col>
              </el-row>
              <el-row :gutter="20">
                <el-col :span="12">
                  <el-form-item v-if="form.portscanTool === 'naabu'" label="扫描类型">
                    <el-radio-group v-model="form.scanType">
                      <el-radio label="c">CONNECT</el-radio>
                      <el-radio label="s">SYN</el-radio>
                    </el-radio-group>
                  </el-form-item>
                </el-col>
                <el-col :span="12">
                  <el-form-item label="超时(秒)">
                    <el-input-number v-model="form.portscanTimeout" :min="5" :max="1200" style="width:100%" />
                  </el-form-item>
                </el-col>
              </el-row>
              <el-form-item label="高级选项">
                <el-checkbox v-model="form.skipHostDiscovery">跳过主机发现 (-Pn)</el-checkbox>
                <span class="form-hint">跳过主机存活检测，直接扫描端口</span>
              </el-form-item>
            </template>
          </el-collapse-item>

          <!-- 端口识别 -->
          <el-collapse-item name="portidentify">
            <template #title>
              <span class="collapse-title">端口识别 <el-tag v-if="form.portidentifyEnable" type="success" size="small">开</el-tag></span>
            </template>
            <el-form-item label="启用">
              <el-switch v-model="form.portidentifyEnable" />
            </el-form-item>
            <template v-if="form.portidentifyEnable">
              <el-form-item label="超时(秒)">
                <el-input-number v-model="form.portidentifyTimeout" :min="5" :max="300" />
                <span class="form-hint">单个主机超时时间</span>
              </el-form-item>
              <el-form-item label="Nmap参数">
                <el-input v-model="form.portidentifyArgs" placeholder="-sV --version-intensity 5" />
              </el-form-item>
            </template>
          </el-collapse-item>

          <!-- 指纹识别 -->
          <el-collapse-item name="fingerprint">
            <template #title>
              <span class="collapse-title">指纹识别 <el-tag v-if="form.fingerprintEnable" type="success" size="small">开</el-tag></span>
            </template>
            <el-form-item label="启用">
              <el-switch v-model="form.fingerprintEnable" />
            </el-form-item>
            <template v-if="form.fingerprintEnable">
              <el-form-item label="探测工具">
                <el-radio-group v-model="form.fingerprintTool">
                  <el-radio label="httpx">Httpx (推荐)</el-radio>
                  <el-radio label="builtin">Wappalyzer (内置)</el-radio>
                </el-radio-group>
              </el-form-item>
              <el-form-item label="附加功能">
                <el-checkbox v-model="form.fingerprintIconHash">Icon Hash</el-checkbox>
                <el-checkbox v-model="form.fingerprintCustomEngine">自定义指纹</el-checkbox>
                <el-checkbox v-model="form.fingerprintScreenshot">网页截图</el-checkbox>
              </el-form-item>
              <el-row :gutter="20">
                <el-col :span="12">
                  <el-form-item label="超时(秒)">
                    <el-input-number v-model="form.fingerprintTimeout" :min="5" :max="120" style="width:100%" />
                    <span class="form-hint">并发数由Worker设置控制</span>
                  </el-form-item>
                </el-col>
              </el-row>
            </template>
          </el-collapse-item>

          <!-- 漏洞扫描 -->
          <el-collapse-item name="pocscan">
            <template #title>
              <span class="collapse-title">漏洞扫描 <el-tag v-if="form.pocscanEnable" type="success" size="small">开</el-tag></span>
            </template>
            <el-form-item label="启用">
              <el-switch v-model="form.pocscanEnable" />
              <span class="form-hint">使用 Nuclei 引擎</span>
            </el-form-item>
            <template v-if="form.pocscanEnable">
              <el-form-item label="自动扫描">
                <el-checkbox v-model="form.pocscanAutoScan" :disabled="form.pocscanCustomOnly">自定义标签映射</el-checkbox>
                <el-checkbox v-model="form.pocscanAutomaticScan" :disabled="form.pocscanCustomOnly">Web指纹自动匹配</el-checkbox>
              </el-form-item>
              <el-form-item label="自定义POC">
                <el-checkbox v-model="form.pocscanCustomOnly">只使用自定义POC</el-checkbox>
              </el-form-item>
              <el-form-item label="严重级别">
                <el-checkbox-group v-model="form.pocscanSeverity">
                  <el-checkbox label="critical">Critical</el-checkbox>
                  <el-checkbox label="high">High</el-checkbox>
                  <el-checkbox label="medium">Medium</el-checkbox>
                  <el-checkbox label="low">Low</el-checkbox>
                  <el-checkbox label="info">Info</el-checkbox>
                </el-checkbox-group>
              </el-form-item>
              <el-form-item label="目标超时">
                <el-input-number v-model="form.pocscanTargetTimeout" :min="30" :max="600" />
                <span class="form-hint">秒</span>
              </el-form-item>
            </template>
          </el-collapse-item>

          <!-- 高级设置 -->
          <el-collapse-item name="advanced">
            <template #title>
              <span class="collapse-title">高级设置</span>
            </template>
            <el-form-item label="任务拆分">
              <el-input-number v-model="form.batchSize" :min="0" :max="1000" :step="10" />
              <span class="form-hint">每批目标数量，0=不拆分</span>
            </el-form-item>
          </el-collapse-item>
        </el-collapse>

        <!-- 操作按钮 -->
        <div class="form-actions">
          <el-button type="primary" :loading="submitting" @click="handleSubmit">{{ isEdit ? '保存' : '创建任务' }}</el-button>
          <el-button @click="handleCancel">取消</el-button>
        </div>
      </el-form>
    </el-card>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { createTask, updateTask, getTaskDetail, startTask, getWorkerList, getScanConfig, saveScanConfig } from '@/api/task'
import { useWorkspaceStore } from '@/stores/workspace'
import request from '@/api/request'

const router = useRouter()
const route = useRoute()
const workspaceStore = useWorkspaceStore()
const formRef = ref()
const submitting = ref(false)
const workspaces = ref([])
const organizations = ref([])
const workers = ref([])
const activeCollapse = ref(['portscan', 'fingerprint'])
const isEdit = ref(false)

const form = reactive({
  id: '',
  name: '',
  target: '',
  workspaceId: '',
  orgId: '',
  isCron: false,
  cronRule: '',
  workers: [],
  batchSize: 50,
  // 子域名扫描
  domainscanEnable: false,
  domainscanSubfinder: true,
  domainscanTimeout: 300,
  domainscanMaxEnumTime: 10,
  domainscanThreads: 10,
  domainscanRateLimit: 0,
  domainscanRemoveWildcard: true,
  domainscanResolveDNS: true,
  domainscanConcurrent: 50,
  // 端口扫描
  portscanEnable: true,
  portscanTool: 'naabu',
  portscanRate: 1000,
  ports: 'top100',
  portThreshold: 100,
  scanType: 'c',
  portscanTimeout: 60,
  skipHostDiscovery: false,
  // 端口识别
  portidentifyEnable: false,
  portidentifyTimeout: 30,
  portidentifyArgs: '',
  // 指纹识别
  fingerprintEnable: true,
  fingerprintTool: 'httpx',
  fingerprintIconHash: true,
  fingerprintCustomEngine: false,
  fingerprintScreenshot: false,
  fingerprintTimeout: 30,
  // 漏洞扫描
  pocscanEnable: false,
  pocscanAutoScan: true,
  pocscanAutomaticScan: true,
  pocscanCustomOnly: false,
  pocscanSeverity: ['critical', 'high', 'medium'],
  pocscanTargetTimeout: 600
})

const rules = {
  name: [{ required: true, message: '请输入任务名称', trigger: 'blur' }],
  target: [{ required: true, message: '请输入扫描目标', trigger: 'blur' }]
}

onMounted(async () => {
  await loadWorkspaces()
  await loadOrganizations()
  await loadWorkers()
  
  // 检查是否是编辑模式
  if (route.query.id) {
    isEdit.value = true
    await loadTaskDetail(route.query.id)
  } else {
    // 加载用户上次保存的扫描配置
    try {
      const res = await getScanConfig()
      if (res.code === 0 && res.config) {
        const config = JSON.parse(res.config)
        applyConfig(config)
      }
    } catch (e) { console.error('加载扫描配置失败:', e) }
    
    // 设置默认工作空间
    let wsId = workspaceStore.currentWorkspaceId
    if (wsId === 'all' || !wsId) {
      const defaultWs = workspaces.value.find(ws => ws.name === '默认工作空间')
      wsId = defaultWs ? defaultWs.id : (workspaces.value.length > 0 ? workspaces.value[0].id : '')
    }
    form.workspaceId = wsId
  }
})

async function loadWorkspaces() {
  try {
    const res = await request.post('/workspace/list', { page: 1, pageSize: 100 })
    if (res.code === 0) workspaces.value = res.list || []
  } catch (e) { console.error(e) }
}

async function loadOrganizations() {
  try {
    const res = await request.post('/organization/list', { page: 1, pageSize: 100 })
    if (res.code === 0) organizations.value = (res.list || []).filter(org => org.status === 'enable')
  } catch (e) { console.error(e) }
}

async function loadWorkers() {
  try {
    const res = await getWorkerList()
    const data = res.data || res
    if (data.code === 0) workers.value = (data.list || []).filter(w => w.status === 'running')
  } catch (e) { console.error(e) }
}

async function loadTaskDetail(taskId) {
  try {
    const res = await getTaskDetail({ id: taskId })
    if (res.code === 0 && res.data) {
      Object.assign(form, res.data)
      if (res.data.config) {
        const config = JSON.parse(res.data.config)
        applyConfig(config)
      }
    }
  } catch (e) { console.error(e) }
}

function applyConfig(config) {
  Object.assign(form, {
    batchSize: config.batchSize || 50,
    // 子域名扫描
    domainscanEnable: config.domainscan?.enable ?? false,
    domainscanSubfinder: config.domainscan?.subfinder ?? true,
    domainscanTimeout: config.domainscan?.timeout || 300,
    domainscanMaxEnumTime: config.domainscan?.maxEnumerationTime || 10,
    domainscanThreads: config.domainscan?.threads || 10,
    domainscanRateLimit: config.domainscan?.rateLimit || 0,
    domainscanRemoveWildcard: config.domainscan?.removeWildcard ?? true,
    domainscanResolveDNS: config.domainscan?.resolveDNS ?? true,
    domainscanConcurrent: config.domainscan?.concurrent || 50,
    // 端口扫描
    portscanEnable: config.portscan?.enable ?? true,
    portscanTool: config.portscan?.tool || 'naabu',
    portscanRate: config.portscan?.rate || 1000,
    ports: config.portscan?.ports || 'top100',
    portThreshold: config.portscan?.portThreshold || 100,
    scanType: config.portscan?.scanType || 'c',
    portscanTimeout: config.portscan?.timeout || 60,
    skipHostDiscovery: config.portscan?.skipHostDiscovery ?? false,
    // 端口识别
    portidentifyEnable: config.portidentify?.enable ?? false,
    portidentifyTimeout: config.portidentify?.timeout || 30,
    portidentifyArgs: config.portidentify?.args || '',
    // 指纹识别
    fingerprintEnable: config.fingerprint?.enable ?? true,
    fingerprintTool: config.fingerprint?.tool || (config.fingerprint?.httpx ? 'httpx' : 'builtin'),
    fingerprintIconHash: config.fingerprint?.iconHash ?? true,
    fingerprintCustomEngine: config.fingerprint?.customEngine ?? false,
    fingerprintScreenshot: config.fingerprint?.screenshot ?? false,
    fingerprintTimeout: config.fingerprint?.targetTimeout || 30,
    // 漏洞扫描
    pocscanEnable: config.pocscan?.enable ?? false,
    pocscanAutoScan: config.pocscan?.autoScan ?? true,
    pocscanAutomaticScan: config.pocscan?.automaticScan ?? true,
    pocscanCustomOnly: config.pocscan?.customPocOnly ?? false,
    pocscanSeverity: config.pocscan?.severity ? config.pocscan.severity.split(',') : ['critical', 'high', 'medium'],
    pocscanTargetTimeout: config.pocscan?.targetTimeout || 600
  })
}

// 防抖保存配置
let saveConfigTimer = null
function debounceSaveConfig() {
  if (saveConfigTimer) clearTimeout(saveConfigTimer)
  saveConfigTimer = setTimeout(() => {
    const config = buildConfig()
    saveScanConfig({ config: JSON.stringify(config) }).catch(e => console.error('自动保存配置失败:', e))
  }, 500)
}

// 监听扫描配置变化，自动保存（仅在新建任务时）
// 使用 getter 函数返回配置字段的快照
watch(
  () => JSON.stringify({
    batchSize: form.batchSize,
    domainscanEnable: form.domainscanEnable,
    domainscanSubfinder: form.domainscanSubfinder,
    domainscanTimeout: form.domainscanTimeout,
    domainscanMaxEnumTime: form.domainscanMaxEnumTime,
    domainscanThreads: form.domainscanThreads,
    domainscanRateLimit: form.domainscanRateLimit,
    domainscanRemoveWildcard: form.domainscanRemoveWildcard,
    domainscanResolveDNS: form.domainscanResolveDNS,
    domainscanConcurrent: form.domainscanConcurrent,
    portscanEnable: form.portscanEnable,
    portscanTool: form.portscanTool,
    portscanRate: form.portscanRate,
    ports: form.ports,
    portThreshold: form.portThreshold,
    scanType: form.scanType,
    portscanTimeout: form.portscanTimeout,
    skipHostDiscovery: form.skipHostDiscovery,
    portidentifyEnable: form.portidentifyEnable,
    portidentifyTimeout: form.portidentifyTimeout,
    portidentifyArgs: form.portidentifyArgs,
    fingerprintEnable: form.fingerprintEnable,
    fingerprintTool: form.fingerprintTool,
    fingerprintIconHash: form.fingerprintIconHash,
    fingerprintCustomEngine: form.fingerprintCustomEngine,
    fingerprintScreenshot: form.fingerprintScreenshot,
    fingerprintTimeout: form.fingerprintTimeout,
    pocscanEnable: form.pocscanEnable,
    pocscanAutoScan: form.pocscanAutoScan,
    pocscanAutomaticScan: form.pocscanAutomaticScan,
    pocscanCustomOnly: form.pocscanCustomOnly,
    pocscanSeverity: form.pocscanSeverity,
    pocscanTargetTimeout: form.pocscanTargetTimeout
  }),
  () => {
    if (!isEdit.value) {
      debounceSaveConfig()
    }
  }
)

function buildConfig() {
  return {
    batchSize: form.batchSize,
    domainscan: {
      enable: form.domainscanEnable,
      subfinder: form.domainscanSubfinder,
      timeout: form.domainscanTimeout,
      maxEnumerationTime: form.domainscanMaxEnumTime,
      threads: form.domainscanThreads,
      rateLimit: form.domainscanRateLimit,
      removeWildcard: form.domainscanRemoveWildcard,
      resolveDNS: form.domainscanResolveDNS,
      concurrent: form.domainscanConcurrent
    },
    portscan: {
      enable: form.portscanEnable,
      tool: form.portscanTool,
      rate: form.portscanRate,
      ports: form.ports,
      portThreshold: form.portThreshold,
      scanType: form.scanType,
      timeout: form.portscanTimeout,
      skipHostDiscovery: form.skipHostDiscovery
    },
    portidentify: {
      enable: form.portidentifyEnable,
      timeout: form.portidentifyTimeout,
      args: form.portidentifyArgs
    },
    fingerprint: {
      enable: form.fingerprintEnable,
      tool: form.fingerprintTool,
      iconHash: form.fingerprintIconHash,
      customEngine: form.fingerprintCustomEngine,
      screenshot: form.fingerprintScreenshot,
      targetTimeout: form.fingerprintTimeout
    },
    pocscan: {
      enable: form.pocscanEnable,
      useNuclei: true,
      autoScan: form.pocscanAutoScan,
      automaticScan: form.pocscanAutomaticScan,
      customPocOnly: form.pocscanCustomOnly,
      severity: form.pocscanSeverity.join(','),
      targetTimeout: form.pocscanTargetTimeout
    }
  }
}

async function handleSubmit() {
  try {
    await formRef.value.validate()
  } catch (e) { return }

  submitting.value = true
  try {
    const config = buildConfig()
    const params = {
      name: form.name,
      target: form.target,
      workspaceId: form.workspaceId,
      orgId: form.orgId,
      isCron: form.isCron,
      cronRule: form.cronRule,
      workers: form.workers,
      config: JSON.stringify(config)
    }

    let res
    if (isEdit.value) {
      params.id = form.id
      res = await updateTask(params)
    } else {
      res = await createTask(params)
    }

    if (res.code === 0) {
      ElMessage.success(isEdit.value ? '任务更新成功' : '任务创建成功')
      if (!isEdit.value && res.id) {
        await startTask({ id: res.id })
        ElMessage.success('任务已启动')
      }
      router.push('/task')
    } else {
      ElMessage.error(res.msg || '操作失败')
    }
  } finally {
    submitting.value = false
  }
}

function handleCancel() {
  router.push('/task')
}
</script>

<style lang="scss" scoped>
.task-create-page {
  .create-card {
    .task-form {
      padding: 20px 40px;
    }
  }

  .config-collapse {
    margin: 20px 0;

    :deep(.el-collapse-item__header) {
      background: var(--el-fill-color-light);
      padding: 0 16px;
      font-size: 14px;
      font-weight: 500;
      height: 44px;
      line-height: 44px;

      &:hover {
        background: var(--el-fill-color);
      }
    }

    :deep(.el-collapse-item__wrap) {
      border: none;
    }

    :deep(.el-collapse-item__content) {
      padding: 20px 16px;
    }

    .collapse-title {
      display: flex;
      align-items: center;
      gap: 10px;
    }
  }

  .form-hint {
    margin-left: 10px;
    color: var(--el-text-color-secondary);
    font-size: 12px;
  }

  .form-actions {
    margin-top: 30px;
    padding-top: 20px;
    border-top: 1px solid var(--el-border-color-lighter);

    .el-button {
      min-width: 100px;
    }
  }
}
</style>
