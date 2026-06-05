import { useState, useEffect, useCallback, useRef } from 'react'
import { ChevronDown, ChevronRight, Play, Terminal, RotateCcw, X, Search, RefreshCw, Zap } from 'lucide-react'
import { api } from '../api/client'

function buildCommandList(data) {
  if (!data) return []
  const cmds = []
  for (const cat of data.categories) {
    for (const t of cat.targets) {
      if (t.kind === 'set') {
        cmds.push({ text: `SET ${t.target}:`, verb: 'SET', target: t.target, kind: 'set' })
        cmds.push({ text: `LST ${t.target}`, verb: 'LST', target: t.target, kind: 'set' })
      } else if (t.kind === 'add') {
        cmds.push({ text: `ADD ${t.target}:`, verb: 'ADD', target: t.target, kind: 'add' })
        cmds.push({ text: `LST ${t.target}`, verb: 'LST', target: t.target, kind: 'add' })
        cmds.push({ text: `MOD ${t.target}:`, verb: 'MOD', target: t.target, kind: 'add' })
        cmds.push({ text: `RMV ${t.target}:`, verb: 'RMV', target: t.target, kind: 'add' })
      }
    }
  }
  for (const act of (data.actions || [])) {
    cmds.push({ text: `ACT ${act.action}`, verb: 'ACT', action: act.action })
  }
  return cmds
}

function parseCommand(input, data) {
  if (!data) return { verb: null, target: null, rest: '' }
  const trimmed = input.trim()
  if (!trimmed) return { verb: null, target: null, rest: '' }

  const parts = trimmed.split(/\s+/)
  const verb = parts[0]?.toUpperCase()
  if (parts.length === 1) return { verb, target: null, rest: '' }

  if (verb === 'ACT') {
    return { verb, target: parts[1]?.toUpperCase(), rest: parts.slice(2).join(' ') }
  }

  const targetPart = parts[1] || ''
  const colonIdx = targetPart.indexOf(':')
  const target = colonIdx >= 0 ? targetPart.slice(0, colonIdx).toUpperCase() : targetPart.toUpperCase()
  const afterColon = colonIdx >= 0
    ? (targetPart.slice(colonIdx + 1) + ' ' + parts.slice(2).join(' ')).trim()
    : parts.slice(2).join(' ')

  return { verb, target, rest: afterColon }
}

function findTarget(data, target) {
  if (!data) return null
  for (const cat of data.categories) {
    for (const t of cat.targets) {
      if (t.target === target) return t
    }
  }
  return null
}

export default function ConfigPage() {
  const [data, setData] = useState(null)
  const [command, setCommand] = useState('')
  const [result, setResult] = useState(null)
  const [loading, setLoading] = useState(false)
  const [suggestions, setSuggestions] = useState([])
  const [showSuggestions, setShowSuggestions] = useState(false)
  const [selectedIdx, setSelectedIdx] = useState(-1)
  const [confirmAction, setConfirmAction] = useState(null)
  const [formMode, setFormMode] = useState(null)
  const [formTarget, setFormTarget] = useState(null)
  const [formValues, setFormValues] = useState({})
  const [modId, setModId] = useState(null)
  const inputRef = useRef(null)
  const listRef = useRef(null)

  const fetchData = useCallback(async () => {
    try {
      const d = await api.getConfigTargets()
      setData(d)
    } catch (e) {
      setResult({ ok: false, error: e.message })
    }
  }, [])

  useEffect(() => { fetchData() }, [fetchData])

  // ---- autocomplete ----

  const computeSuggestions = useCallback((input, d) => {
    if (!d || !input.trim()) return []
    const upper = input.toUpperCase().trim()
    const all = buildCommandList(d)

    const prefix = all.filter(c => c.text.startsWith(upper)).slice(0, 6)
    if (prefix.length > 0) return prefix

    const tokens = upper.split(/\s+/)
    if (tokens.length <= 2) {
      const last = tokens[tokens.length - 1]
      const prefix2 = tokens.length > 1 ? tokens.slice(0, -1).join(' ') + ' ' : ''
      const verbs = ['SET', 'ADD', 'LST', 'MOD', 'RMV', 'ACT']
        .filter(v => v.startsWith(last) && v !== last)
      if (verbs.length > 0) return verbs.map(v => ({ text: prefix2 + v, verb: v }))
      if (tokens.length === 2) {
        const verb = tokens[0]
        if (['SET', 'ADD', 'LST', 'MOD', 'RMV'].includes(verb)) {
          for (const cat of d.categories) {
            for (const t of cat.targets) {
              if (t.target.startsWith(last)) return [{ text: `${verb} ${t.target}`, verb, target: t.target, kind: t.kind }]
            }
          }
        }
        if (verb === 'ACT') {
          for (const act of (d.actions || [])) {
            if (act.action.startsWith(last)) return [{ text: `ACT ${act.action}`, verb: 'ACT', action: act.action }]
          }
        }
      }
    }
    return []
  }, [])

  const handleCommandChange = (value) => {
    setCommand(value)
    const suggs = computeSuggestions(value, data)
    setSuggestions(suggs)
    setShowSuggestions(suggs.length > 0)
    setSelectedIdx(-1)
    setResult(null)
    detectMode(value, data)
  }

  const selectSuggestion = (s) => {
    setCommand(s.text)
    setSuggestions([])
    setShowSuggestions(false)
    setResult(null)
    detectMode(s.text, data)
    inputRef.current?.focus()
  }

  const handleKeyDown = (e) => {
    if (!showSuggestions) return
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      setSelectedIdx(i => Math.min(i + 1, suggestions.length - 1))
    } else if (e.key === 'ArrowUp') {
      e.preventDefault()
      setSelectedIdx(i => Math.max(i - 1, 0))
    } else if (e.key === 'Enter' && selectedIdx >= 0) {
      e.preventDefault()
      selectSuggestion(suggestions[selectedIdx])
    } else if (e.key === 'Escape') {
      setShowSuggestions(false)
    }
  }

  // ---- mode detection from command text ----

  const detectMode = (cmd, d) => {
    if (!d || !cmd.trim()) {
      setFormMode(null); setFormTarget(null); setFormValues({}); return
    }
    const { verb, target, rest } = parseCommand(cmd, d)
    if (!verb || !target) { setFormMode(null); setFormTarget(null); return }

    const t = findTarget(d, target)
    if (!t) { setFormMode(null); setFormTarget(null); return }

    setFormTarget(target)

    if (verb === 'LST') {
      setFormMode('lst')
      setFormValues({})
      executeLst(target)
    } else if (verb === 'SET' && t.kind === 'set') {
      setFormMode('set')
      const vals = { ...(t.value || {}) }
      setFormValues(vals)
      setModId(null)
      // rebuild command with current values
      const parts = Object.entries(vals).filter(([,v]) => v).map(([k,v]) => `${k}=${v}`)
      setCommand(`SET ${target}: ${parts.join(', ')}`)
    } else if (verb === 'ADD' && t.kind === 'add') {
      setFormMode('add')
      const defs = {}
      for (const f of (t.fields || [])) {
        defs[f.key] = f.type === 'select' ? (f.options?.[0] || '') : ''
      }
      setFormValues(defs)
      setModId(null)
    } else if (verb === 'MOD' && t.kind === 'add') {
      setFormMode('mod')
      const idMatch = rest.match(/ID=(\d+)/i)
      if (idMatch) {
        const id = parseInt(idMatch[1])
        setModId(id)
        const inst = t.instances?.find(i => i.id === id)
        setFormValues(inst ? { ...inst.attrs } : {})
      } else {
        setFormValues({})
      }
    } else if (verb === 'RMV' && t.kind === 'add') {
      setFormMode('rmv')
      const idMatch = rest.match(/ID=(\d+)/i)
      if (idMatch) setModId(parseInt(idMatch[1]))
      setFormValues({})
    } else {
      setFormMode(null)
    }
  }

  const executeLst = async (target) => {
    const cmd = `LST ${target}`
    setLoading(true)
    try {
      const r = await api.execConfigCommand(cmd)
      setResult(r)
    } catch (e) {
      setResult({ ok: false, error: e.message })
    } finally {
      setLoading(false)
    }
  }

  // ---- left panel click ----

  const handleLeftClick = (target, kind) => {
    let cmd
    if (kind === 'set') {
      const t = findTarget(data, target)
      const fields = t?.value
        ? Object.entries(t.value).filter(([,v]) => v).map(([k, v]) => `${k}=${v}`).join(', ')
        : ''
      cmd = `SET ${target}: ${fields}`
    } else {
      cmd = `LST ${target}`
    }
    setCommand(cmd)
    setSuggestions([])
    setShowSuggestions(false)
    setResult(null)
    detectMode(cmd, data)
    inputRef.current?.focus()
  }

  // ---- form updates ----

  const updateFormValue = (key, value) => {
    const next = { ...formValues, [key]: value }
    setFormValues(next)
    rebuildCommand(next)
  }

  const rebuildCommand = (values) => {
    if (!formTarget) return
    const verb = formMode === 'add' ? 'ADD' : formMode === 'mod' ? 'MOD' : 'SET'
    let cmd = `${verb} ${formTarget}:`
    const parts = []
    if (formMode === 'mod' && modId) parts.push(`ID=${modId}`)
    for (const [k, v] of Object.entries(values)) {
      if (v !== undefined && v !== '') parts.push(`${k}=${v}`)
    }
    cmd += ' ' + parts.join(', ')
    setCommand(cmd)
    return cmd
  }

  // ---- execute ----

  const handleExec = async () => {
    if (!command.trim()) return

    let cmd = command.trim()

    // validate: SET/ADD/MOD require at least one KEY=VALUE pair
    if (formMode && formMode !== 'lst' && formMode !== 'rmv') {
      const hasField = Object.values(formValues).some(v => v !== undefined && v !== '')
      if (!hasField) {
        setResult({ ok: false, error: 'Please fill at least one field' })
        return
      }
      cmd = rebuildCommand(formValues) || cmd
    }

    // Check for confirm actions
    const actMatch = cmd.match(/^ACT\s+(\S+)/i)
    if (actMatch) {
      const action = actMatch[1].toUpperCase()
      const act = data?.actions?.find(a => a.action === action)
      if (act?.confirm) {
        setConfirmAction({ action, label: act.label, confirm: act.confirm })
        return
      }
    }

    await execCmd(cmd)
  }

  const execCmd = async (cmd) => {
    setLoading(true)
    try {
      const r = await api.execConfigCommand(cmd)
      setResult(r)
      if (r.ok) {
        await fetchData()
        if (formMode === 'lst' && formTarget) {
          executeLst(formTarget)
        } else {
          setFormMode(null)
          setFormTarget(null)
          setFormValues({})
          setCommand('')
        }
      }
    } catch (e) {
      setResult({ ok: false, error: e.message })
    } finally {
      setLoading(false)
      setConfirmAction(null)
    }
  }

  const execAction = async (action) => {
    await execCmd(`ACT ${action}`)
  }

  // ---- range extra field for LGFAILFIBPLCY ----

  const getRangeExtra = () => {
    if (formTarget !== 'LGFAILFIBPLCY') return null
    const range = (formValues['RANGE'] || '').split(':')[0]
    if (range === 'SINGLE_USER') return { key: 'RANGE_VAL', label: 'Username', type: 'text', placeholder: 'testuser' }
    if (range === 'IP') return { key: 'RANGE_VAL', label: 'IP Address', type: 'text', placeholder: '192.168.1.1' }
    return null
  }

  // ---- render ----

  if (!data) return <div className="card">Loading...</div>

  const t = formTarget ? findTarget(data, formTarget) : null

  return (
    <div>
      {confirmAction && (
        <div className="modal-overlay" onClick={() => setConfirmAction(null)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <h2>{confirmAction.label}</h2>
            <p style={{ color: 'var(--text-secondary)', fontSize: 13, marginBottom: 16 }}>
              {confirmAction.confirm}
            </p>
            <div className="modal-actions">
              <button className="btn btn-ghost" onClick={() => setConfirmAction(null)}>Cancel</button>
              <button className="btn btn-danger" onClick={() => execAction(confirmAction.action)}>Confirm</button>
            </div>
          </div>
        </div>
      )}

      <div className="page-header">
        <h1>Runtime Config</h1>
      </div>

      <div style={{ display: 'flex', gap: 20, minHeight: 'calc(100vh - 180px)' }}>
        {/* Left panel */}
        <div style={{ width: 260, flexShrink: 0 }}>
          <div className="card" style={{ padding: 12 }}>
            {data.categories.map(cat => (
              <div key={cat.name}>
                <div style={{
                  display: 'flex', alignItems: 'center', gap: 6,
                  padding: '8px 4px', fontWeight: 600, fontSize: 13,
                  borderBottom: '1px solid var(--border)',
                  color: 'var(--text-secondary)',
                }}>
                  <Terminal size={13} />{cat.name}
                </div>
                {cat.targets.map(t => (
                  <div
                    key={t.target}
                    onClick={() => handleLeftClick(t.target, t.kind)}
                    style={{
                      display: 'flex', alignItems: 'center', gap: 8, padding: '8px 4px 8px 16px',
                      cursor: 'pointer', fontSize: 13, borderRadius: 4,
                      color: formTarget === t.target ? 'var(--primary)' : 'var(--text)',
                      background: formTarget === t.target ? '#e8f0fe' : 'transparent',
                    }}
                  >
                    <span style={{ fontFamily: 'monospace', fontSize: 12, color: 'var(--text-secondary)', minWidth: 50 }}>
                      {t.kind === 'set' ? 'SET' : 'ADD'}
                    </span>
                    <span style={{ flex: 1 }}>{t.label}</span>
                    <span style={{ fontFamily: 'monospace', fontSize: 10, color: 'var(--text-muted)' }}>
                      {t.target}
                    </span>
                  </div>
                ))}
              </div>
            ))}

            <div style={{ marginTop: 12 }}>
              <div style={{
                fontWeight: 600, fontSize: 13, padding: '8px 4px',
                borderBottom: '1px solid var(--border)', color: 'var(--text-secondary)',
                display: 'flex', alignItems: 'center', gap: 6,
              }}>
                <Zap size={13} />Actions
              </div>
              {data.actions?.map(act => (
                <div
                  key={act.action}
                  onClick={() => {
                    setCommand(`ACT ${act.action}`)
                    setSuggestions([])
                    setShowSuggestions(false)
                    setFormMode(null)
                    setFormTarget(null)
                    inputRef.current?.focus()
                  }}
                  style={{
                    padding: '8px 4px 8px 20px', cursor: 'pointer', fontSize: 13,
                    color: 'var(--text-secondary)', display: 'flex', alignItems: 'center', gap: 8,
                  }}
                >
                  {act.action === 'SYSTEMRST' ? <RotateCcw size={12} /> :
                    act.action === 'CLRLIMIT' ? <X size={12} /> :
                      <RefreshCw size={12} />}
                  <span style={{ fontFamily: 'monospace', fontSize: 11 }}>ACT</span> {act.label}
                </div>
              ))}
            </div>
          </div>
        </div>

        {/* Right panel */}
        <div style={{ flex: 1, display: 'flex', flexDirection: 'column', gap: 12 }}>
          {/* Command input with autocomplete */}
          <div style={{ position: 'relative' }}>
            <div style={{
              display: 'flex', alignItems: 'center', gap: 0,
              background: '#1e293b', borderRadius: 6, overflow: 'hidden',
            }}>
              <span style={{
                color: '#4ade80', fontFamily: 'monospace', fontSize: 14,
                padding: '0 12px', userSelect: 'none',
              }}>$</span>
              <input
                ref={inputRef}
                type="text"
                value={command}
                onChange={e => handleCommandChange(e.target.value)}
                onKeyDown={handleKeyDown}
                onFocus={() => { if (suggestions.length > 0) setShowSuggestions(true) }}
                onBlur={() => setTimeout(() => setShowSuggestions(false), 150)}
                placeholder="e.g. LST SYSLOG or SET JWT: EXPIRETIME=96"
                spellCheck={false}
                autoComplete="off"
                style={{
                  flex: 1, background: 'transparent', border: 'none', outline: 'none',
                  color: '#e2e8f0', fontFamily: 'monospace', fontSize: 14,
                  padding: '12px 4px',
                }}
              />
              <button
                className="btn btn-primary"
                onClick={handleExec}
                disabled={loading || !command.trim()}
                style={{
                  borderRadius: '0 6px 6px 0', padding: '12px 20px',
                  fontFamily: 'monospace', fontSize: 13,
                }}
              >
                <Play size={14} />
                {loading ? '...' : 'Exec'}
              </button>
            </div>

            {showSuggestions && suggestions.length > 0 && (
              <div
                ref={listRef}
                style={{
                  position: 'absolute', top: '100%', left: 0, right: 0,
                  background: 'var(--bg)', border: '1px solid var(--border)',
                  borderRadius: 6, marginTop: 4, zIndex: 100,
                  boxShadow: '0 4px 16px rgba(0,0,0,0.12)', overflow: 'hidden',
                }}
              >
                {suggestions.map((s, i) => (
                  <div
                    key={s.text}
                    onMouseDown={() => selectSuggestion(s)}
                    style={{
                      padding: '10px 16px', cursor: 'pointer', fontSize: 13,
                      fontFamily: 'monospace',
                      background: i === selectedIdx ? '#e8f0fe' : 'transparent',
                      color: i === selectedIdx ? 'var(--primary)' : 'var(--text)',
                      borderBottom: i < suggestions.length - 1 ? '1px solid var(--border)' : 'none',
                      display: 'flex', alignItems: 'center', gap: 8,
                    }}
                  >
                    <span style={{ color: 'var(--primary)', fontWeight: 600, minWidth: 48 }}>
                      {s.verb || ''}
                    </span>
                    <span>{s.text}</span>
                  </div>
                ))}
              </div>
            )}
          </div>

          {/* Form panel */}
          {formMode && formMode !== 'lst' && t && (
            <div className="card">
              <div style={{
                display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 16,
              }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                  <span style={{
                    background: 'var(--primary)', color: '#fff', padding: '2px 10px',
                    borderRadius: 4, fontSize: 12, fontFamily: 'monospace', fontWeight: 600,
                  }}>
                    {formMode === 'add' ? 'ADD' : formMode === 'mod' ? 'MOD' : formMode === 'rmv' ? 'RMV' : 'SET'}
                  </span>
                  <span style={{ fontWeight: 600, fontSize: 14 }}>{t.label}</span>
                </div>
                <button className="btn btn-sm btn-ghost" onClick={() => {
                  setFormMode(null); setFormTarget(null); setFormValues({}); setCommand('')
                }}>Cancel</button>
              </div>

              <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
                {formMode !== 'rmv' && t.fields?.map(f => (
                  <div key={f.key} style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
                    <label style={{ width: 130, fontSize: 13, fontWeight: 500, color: 'var(--text-secondary)' }}>
                      {f.label}
                    </label>
                    {f.type === 'select' ? (
                      <select
                        value={formValues[f.key] || ''}
                        onChange={e => updateFormValue(f.key, e.target.value)}
                        style={{
                          flex: 1, padding: '7px 10px', border: '1px solid var(--border)',
                          borderRadius: 'var(--radius)', fontSize: 13, background: 'var(--bg)',
                        }}
                      >
                        {f.options?.map(o => <option key={o} value={o}>{o}</option>)}
                      </select>
                    ) : f.type === 'number' ? (
                      <input
                        type="number"
                        min={f.min} max={f.max}
                        value={formValues[f.key] || ''}
                        onChange={e => updateFormValue(f.key, e.target.value)}
                        style={{
                          flex: 1, padding: '7px 10px', border: '1px solid var(--border)',
                          borderRadius: 'var(--radius)', fontSize: 13,
                        }}
                      />
                    ) : (
                      <input
                        type="text"
                        value={formValues[f.key] || ''}
                        onChange={e => updateFormValue(f.key, e.target.value)}
                        placeholder={f.placeholder}
                        style={{
                          flex: 1, padding: '7px 10px', border: '1px solid var(--border)',
                          borderRadius: 'var(--radius)', fontSize: 13,
                        }}
                      />
                    )}
                  </div>
                ))}

                {getRangeExtra() && (
                  <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
                    <label style={{ width: 130, fontSize: 13, fontWeight: 500, color: 'var(--text-secondary)' }}>
                      {getRangeExtra().label}
                    </label>
                    <input
                      type="text"
                      value={formValues['RANGE_VAL'] || ''}
                      onChange={e => {
                        const v = e.target.value
                        const base = (formValues['RANGE'] || 'ALL_USER').split(':')[0]
                        const newRange = v ? `${base}:${v}` : base
                        updateFormValue('RANGE', newRange)
                        const next = { ...formValues, RANGE: newRange, RANGE_VAL: v }
                        setFormValues(next)
                      }}
                      placeholder={getRangeExtra().placeholder}
                      style={{
                        flex: 1, padding: '7px 10px', border: '1px solid var(--border)',
                        borderRadius: 'var(--radius)', fontSize: 13,
                      }}
                    />
                  </div>
                )}

                {formMode === 'rmv' && (
                  <div style={{ color: 'var(--danger)', fontSize: 13, marginBottom: 8 }}>
                    This will delete the config entry. This action cannot be undone.
                  </div>
                )}
              </div>
            </div>
          )}

          {/* LST results */}
          {formMode === 'lst' && result && (
            <div className="card" style={{
              background: result.ok ? '#f8fafc' : '#fef2f2',
              borderColor: result.ok ? '#e2e8f0' : '#fecaca',
            }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 8 }}>
                <span style={{
                  background: 'var(--primary)', color: '#fff', padding: '2px 10px',
                  borderRadius: 4, fontSize: 12, fontFamily: 'monospace', fontWeight: 600,
                }}>LST</span>
                <span style={{ fontWeight: 600, fontSize: 14 }}>{t?.label || formTarget}</span>
              </div>
              <pre style={{
                fontSize: 13, whiteSpace: 'pre-wrap', margin: 0, fontFamily: 'monospace',
                color: result.ok ? 'var(--text)' : '#991b1b',
                maxHeight: 300, overflow: 'auto',
              }}>
                {result.output || result.error}
              </pre>
            </div>
          )}

          {/* Execution result */}
          {result && formMode !== 'lst' && (
            <div className="card" style={{
              background: result.ok ? '#f0fdf4' : '#fef2f2',
              borderColor: result.ok ? '#bbf7d0' : '#fecaca',
            }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 4 }}>
                <span style={{
                  color: result.ok ? 'var(--success)' : 'var(--danger)',
                  fontWeight: 600, fontSize: 13,
                }}>
                  {result.ok ? 'Success' : 'Error'}
                </span>
              </div>
              <pre style={{
                fontSize: 13, whiteSpace: 'pre-wrap', margin: 0,
                color: result.ok ? '#166534' : '#991b1b', fontFamily: 'monospace',
              }}>
                {result.output || result.error}
              </pre>
            </div>
          )}

          {/* Empty state */}
          {!formMode && !result && (
            <div style={{
              flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center',
              color: 'var(--text-muted)', fontSize: 14, flexDirection: 'column', gap: 8,
            }}>
              <Terminal size={32} style={{ opacity: 0.3 }} />
              <div>Type a command or select from the left panel</div>
              <div style={{ fontSize: 12, opacity: 0.6 }}>
                SET · ADD · LST · MOD · RMV · ACT
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
