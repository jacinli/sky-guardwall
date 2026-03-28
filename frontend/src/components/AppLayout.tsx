import { FireOutlined, LogoutOutlined, SafetyOutlined, SyncOutlined } from '@ant-design/icons'
import { Avatar, Badge, Button, Layout, Menu, Space, Typography, theme } from 'antd'
import dayjs from 'dayjs'
import relativeTime from 'dayjs/plugin/relativeTime'
import { useEffect, useState } from 'react'
import { Outlet, useLocation, useNavigate } from 'react-router-dom'
import client from '../api/client'
import { useAuthStore } from '../store/auth'

dayjs.extend(relativeTime)

const { Header, Sider, Content } = Layout
const { Text } = Typography

export default function AppLayout() {
  const navigate = useNavigate()
  const location = useLocation()
  const logout = useAuthStore((s) => s.logout)
  const { token: colorToken } = theme.useToken()

  const [lastSynced, setLastSynced] = useState<string | null>(null)
  const [syncError, setSyncError] = useState(false)

  const fetchSyncStatus = () => {
    client
      .get('/iptables/rules')
      .then((res) => {
        const d = res.data.data
        if (d.last_synced_at) setLastSynced(d.last_synced_at)
        setSyncError(!!d.sync_has_error)
      })
      .catch(() => {})
  }

  useEffect(() => {
    fetchSyncStatus()
    const interval = setInterval(fetchSyncStatus, 30_000)
    return () => clearInterval(interval)
  }, [])

  const menuItems = [
    {
      key: '/',
      icon: <FireOutlined />,
      label: 'iptables 规则',
    },
    {
      key: '/managed',
      icon: <SafetyOutlined />,
      label: '我的规则',
    },
  ]

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Sider
        theme="dark"
        width={220}
        style={{ position: 'fixed', left: 0, top: 0, bottom: 0, zIndex: 100 }}
      >
        <div
          style={{
            padding: '20px 16px 16px',
            display: 'flex',
            alignItems: 'center',
            gap: 8,
          }}
        >
          <Avatar
            size={36}
            style={{ background: colorToken.colorPrimary, flexShrink: 0 }}
            icon={<FireOutlined />}
          />
          <Text strong style={{ color: '#fff', fontSize: 15 }}>
            SkyGuardwall
          </Text>
        </div>

        <Menu
          theme="dark"
          mode="inline"
          selectedKeys={[location.pathname]}
          items={menuItems}
          onClick={({ key }) => navigate(key)}
          style={{ borderRight: 0 }}
        />
      </Sider>

      <Layout style={{ marginLeft: 220 }}>
        <Header
          style={{
            background: colorToken.colorBgContainer,
            padding: '0 24px',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            borderBottom: `1px solid ${colorToken.colorBorderSecondary}`,
            position: 'sticky',
            top: 0,
            zIndex: 99,
          }}
        >
          <Space size="small">
            <SyncOutlined spin={false} style={{ color: colorToken.colorTextSecondary }} />
            <Text type="secondary" style={{ fontSize: 13 }}>
              {lastSynced
                ? `上次同步 ${dayjs(lastSynced).fromNow()}`
                : '同步中...'}
            </Text>
            {syncError && (
              <Badge status="error" text={<Text type="danger" style={{ fontSize: 12 }}>同步错误</Text>} />
            )}
          </Space>

          <Button
            type="text"
            icon={<LogoutOutlined />}
            onClick={() => {
              logout()
              navigate('/login')
            }}
          >
            退出
          </Button>
        </Header>

        <Content style={{ padding: 24, background: colorToken.colorBgLayout }}>
          <Outlet />
        </Content>
      </Layout>
    </Layout>
  )
}
