<template>
  <div class="address-page">
    <div class="address-container">
      <div class="page-header">
        <h1>收货地址管理</h1>
        <el-button type="primary" @click="openAddDialog">
          <el-icon><Plus /></el-icon>
          添加新地址
        </el-button>
      </div>

      <!-- 地址列表 -->
      <div class="address-list" v-loading="loading">
        <div v-if="addresses.length === 0" class="empty-address">
          <el-empty description="暂无收货地址" />
        </div>

        <div v-else class="address-cards">
          <div
            v-for="addr in addresses"
            :key="addr.id"
            class="address-card"
            :class="{ 'is-default': addr.is_default }"
          >
            <div class="card-header">
              <span class="receiver">{{ addr.receiver_name }}</span>
              <span class="phone">{{ addr.phone }}</span>
              <el-tag v-if="addr.is_default" type="success" size="small">默认</el-tag>
            </div>
            <div class="card-body">
              <div class="address-detail">
                {{ addr.province }} {{ addr.city }} {{ addr.district }}
              </div>
              <div class="address-detail">
                {{ addr.detail_address }}
              </div>
              <div class="postal-code" v-if="addr.postal_code">
                邮编：{{ addr.postal_code }}
              </div>
            </div>
            <div class="card-footer">
              <el-button link type="primary" @click="openEditDialog(addr)">编辑</el-button>
              <el-button link type="primary" @click="handleSetDefault(addr.id)" v-if="!addr.is_default">设为默认</el-button>
              <el-button link type="danger" @click="handleDelete(addr.id)">删除</el-button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 添加/编辑地址弹窗 -->
    <el-dialog
      v-model="dialogVisible"
      :title="isEdit ? '编辑地址' : '添加新地址'"
      width="500px"
      @close="handleDialogClose"
    >
      <el-form ref="formRef" :model="formData" :rules="rules" label-width="80px">
        <el-form-item label="收货人" prop="receiver_name">
          <el-input v-model="formData.receiver_name" placeholder="请输入收货人姓名" />
        </el-form-item>
        <el-form-item label="手机号" prop="phone">
          <el-input v-model="formData.phone" placeholder="请输入手机号" />
        </el-form-item>
        <el-form-item label="省份" prop="province">
          <el-input v-model="formData.province" placeholder="请输入省份" />
        </el-form-item>
        <el-form-item label="城市" prop="city">
          <el-input v-model="formData.city" placeholder="请输入城市" />
        </el-form-item>
        <el-form-item label="区县" prop="district">
          <el-input v-model="formData.district" placeholder="请输入区县" />
        </el-form-item>
        <el-form-item label="详细地址" prop="detail_address">
          <el-input
            v-model="formData.detail_address"
            type="textarea"
            :rows="2"
            placeholder="请输入详细地址"
          />
        </el-form-item>
        <el-form-item label="邮编" prop="postal_code">
          <el-input v-model="formData.postal_code" placeholder="请输入邮政编码（选填）" />
        </el-form-item>
        <el-form-item>
          <el-checkbox v-model="formData.is_default">设为默认地址</el-checkbox>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSubmit" :loading="submitting">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import {
  getAddressList,
  addAddress,
  updateAddress,
  deleteAddress,
  setDefaultAddress
} from '@/api/address'
import type { AddressItem, AddAddressReq } from '@/types'

const loading = ref(false)
const addresses = ref<AddressItem[]>([])
const dialogVisible = ref(false)
const isEdit = ref(false)
const submitting = ref(false)
const formRef = ref()
const editingId = ref<number | null>(null)

const defaultFormData: AddAddressReq = {
  receiver_name: '',
  phone: '',
  province: '',
  city: '',
  district: '',
  detail_address: '',
  postal_code: '',
  is_default: false
}

const formData = reactive<AddAddressReq>({ ...defaultFormData })

const rules = {
  receiver_name: [{ required: true, message: '请输入收货人', trigger: 'blur' }],
  phone: [{ required: true, message: '请输入手机号', trigger: 'blur' }],
  province: [{ required: true, message: '请输入省份', trigger: 'blur' }],
  city: [{ required: true, message: '请输入城市', trigger: 'blur' }],
  district: [{ required: true, message: '请输入区县', trigger: 'blur' }],
  detail_address: [{ required: true, message: '请输入详细地址', trigger: 'blur' }]
}

const fetchAddresses = async () => {
  loading.value = true
  try {
    const res = await getAddressList()
    addresses.value = res?.addresses || []
  } catch (error) {
    console.error('Failed to fetch addresses:', error)
  } finally {
    loading.value = false
  }
}

const openAddDialog = () => {
  isEdit.value = false
  editingId.value = null
  Object.assign(formData, defaultFormData)
  dialogVisible.value = true
}

const openEditDialog = (addr: AddressItem) => {
  isEdit.value = true
  editingId.value = addr.id
  Object.assign(formData, {
    receiver_name: addr.receiver_name,
    phone: addr.phone,
    province: addr.province,
    city: addr.city,
    district: addr.district,
    detail_address: addr.detail_address,
    postal_code: addr.postal_code || '',
    is_default: addr.is_default
  })
  dialogVisible.value = true
}

const handleDialogClose = () => {
  formRef.value?.resetFields()
}

const handleSubmit = async () => {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return

  submitting.value = true
  try {
    if (isEdit.value && editingId.value) {
      await updateAddress({ id: editingId.value, ...formData })
      ElMessage.success('地址更新成功')
    } else {
      await addAddress(formData)
      ElMessage.success('地址添加成功')
    }
    dialogVisible.value = false
    fetchAddresses()
  } catch (error: any) {
    ElMessage.error(error.response?.data?.msg || error.message || '操作失败')
  } finally {
    submitting.value = false
  }
}

const handleSetDefault = async (id: number) => {
  try {
    await setDefaultAddress(id)
    ElMessage.success('设置成功')
    fetchAddresses()
  } catch (error: any) {
    ElMessage.error(error.response?.data?.msg || error.message || '设置失败')
  }
}

const handleDelete = async (id: number) => {
  try {
    await ElMessageBox.confirm('确定要删除这个地址吗？', '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    })
    await deleteAddress(id)
    ElMessage.success('删除成功')
    fetchAddresses()
  } catch (error: any) {
    // ElMessageBox 取消时会抛出 'cancel' 字符串，axios 错误由 request.ts 统一处理
    if (error !== 'cancel') {
      // 如果是 axios 错误，request.ts 已经显示过错误消息，这里不需要再处理
      // 只有其他未知错误才显示
      if (!error.response) {
        ElMessage.error('删除失败')
      }
    }
  }
}

onMounted(() => {
  fetchAddresses()
})
</script>

<style scoped>
.address-page {
  min-height: 100vh;
  background: #f5f5f5;
  padding: 20px 0;
}

.address-container {
  max-width: 900px;
  margin: 0 auto;
  padding: 0 20px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.page-header h1 {
  font-size: 24px;
  color: #333;
}

.address-list {
  min-height: 200px;
}

.empty-address {
  background: white;
  padding: 40px;
  border-radius: 12px;
  text-align: center;
}

.address-cards {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 16px;
}

.address-card {
  background: white;
  border-radius: 12px;
  padding: 20px;
  border: 2px solid #f0f0f0;
  transition: all 0.3s;
}

.address-card:hover {
  border-color: #1890ff;
}

.address-card.is-default {
  border-color: #52c41a;
  background: #f6ffed;
}

.card-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
  padding-bottom: 12px;
  border-bottom: 1px solid #f0f0f0;
}

.receiver {
  font-size: 16px;
  font-weight: bold;
  color: #333;
}

.phone {
  font-size: 14px;
  color: #666;
}

.card-body {
  margin-bottom: 12px;
}

.address-detail {
  font-size: 14px;
  color: #333;
  line-height: 1.6;
  margin-bottom: 4px;
}

.postal-code {
  font-size: 12px;
  color: #999;
  margin-top: 8px;
}

.card-footer {
  display: flex;
  gap: 12px;
  justify-content: flex-end;
}

@media (max-width: 768px) {
  .address-cards {
    grid-template-columns: 1fr;
  }
}
</style>
