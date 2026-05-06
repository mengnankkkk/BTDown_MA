import { useState } from 'react'
import type { FormEvent } from 'react'

import type { SessionCreatePayload } from '../../../types/sessionCreatePayload'

interface SessionCreateFormProps {
  submitting: boolean
  onSubmit: (payload: SessionCreatePayload) => Promise<void>
}

export function SessionCreateForm({ submitting, onSubmit }: SessionCreateFormProps) {
  const [name, setName] = useState('')
  const [magnetUri, setMagnetUri] = useState('')

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    await onSubmit({ name, magnetUri })
    setMagnetUri('')
  }

  return (
    <form className="panel session-form" onSubmit={handleSubmit}>
      <div className="panel-header">
        <h2>新建播放会话</h2>
        <span>提交磁力链接后立即建立下载会话并持续更新状态</span>
      </div>
      <label>
        <span>会话名称</span>
        <input value={name} onChange={(event) => setName(event.target.value)} placeholder="例如：测试视频" />
      </label>
      <label>
        <span>磁力链接</span>
        <textarea
          value={magnetUri}
          onChange={(event) => setMagnetUri(event.target.value)}
          placeholder="magnet:?xt=urn:btih:..."
          rows={4}
        />
      </label>
      <button type="submit" disabled={submitting || magnetUri.trim() === ''}>
        {submitting ? '创建中...' : '创建会话'}
      </button>
    </form>
  )
}
