/**
 * 分镜 MD 导入解析器
 * 支持从导出 MD 文件中反向解析出分镜数据
 *
 * 兼容三种导出格式：
 *   1. shots/index.vue     - 完整版（含项目/剧本/状态/氛围等）
 *   2. scriptEditor.vue    - 精简版（景别/运镜/视角/时长）
 *   3. createChapter.vue   - 含动作字段版
 */

export interface ParsedShot {
  shotType?: string
  cameraMovement?: string
  angle?: string
  durationMs?: number
  dialogue?: string
  action?: string
  visualDesc?: string
  atmosphere?: string
  audioPrompt?: string
  imageUrl?: string
  imagePrompt?: string
  videoPrompt?: string
  // 仅读取不导入
  _projectName?: string
  _scriptName?: string
  _status?: string
}

/**
 * 表头字段名 → shot 对象 key 映射
 */
const TABLE_FIELD_MAP: Record<string, keyof ParsedShot> = {
  '短剧项目': '_projectName',
  '剧本': '_scriptName',
  '景别': 'shotType',
  '运镜': 'cameraMovement',
  '视角': 'angle',
  '时长(ms)': 'durationMs',
  '时长': 'durationMs',
  '状态': '_status',
}

/**
 * ### 标题 → shot 对象 key 映射
 */
const SECTION_MAP: Record<string, keyof ParsedShot> = {
  '台词/旁白': 'dialogue',
  '台词': 'dialogue',
  '旁白': 'dialogue',
  '画面描述': 'visualDesc',
  '动作': 'action',
  '氛围/环境描述': 'atmosphere',
  '氛围': 'atmosphere',
  '音效/BGM提示词': 'audioPrompt',
  '音效提示词': 'audioPrompt',
  'BGM提示词': 'audioPrompt',
}

/**
 * ### 图片类标题 → key 映射（内容为 ![...](url)）
 */
const IMAGE_SECTION_MAP: Record<string, keyof ParsedShot> = {
  '分镜图': 'imageUrl',
  '绘画Prompt图': 'imagePrompt',
  '视频生成Prompt': 'videoPrompt',
}

/**
 * 解析导出的分镜 MD 内容，返回分镜对象数组
 */
export function parseShotMdContent(mdText: string): ParsedShot[] {
  const shots: ParsedShot[] = []

  // 去掉 # 标题行和 > 导出信息行
  const body = mdText
    .replace(/^# .*\n/gm, '')
    .replace(/^> .*\n/gm, '')
    .replace(/^---\s*$/gm, '')

  // 按 ## 镜头 分割
  const shotBlocks = body.split(/^## 镜头\b/gm).filter((block) => block.trim())

  for (const block of shotBlocks) {
    const shot: ParsedShot = {}

    // 去掉镜头编号（行首的数字/特殊字符）
    const content = block.replace(/^\s*[#\d]+\s*/, '')

    // ---- 解析表格 ----
    const tableRegex = /^\| 字段 \| 内容 \|[\s\S]*?(?=\n\n|\n###|$)/m
    const tableMatch = content.match(tableRegex)
    if (tableMatch) {
      const tableRows = tableMatch[0].split('\n').filter((line) => line.trim())
      for (let i = 1; i < tableRows.length; i++) {
        // 跳过表头和分隔行
        if (tableRows[i].includes('---')) continue
        const cols = tableRows[i].split('|').map((c) => c.trim()).filter(Boolean)
        if (cols.length >= 2) {
          const field = cols[0]
          const value = cols[1]
          const key = TABLE_FIELD_MAP[field]
          if (key) {
            if (key === 'durationMs') {
              const num = parseInt(value.replace(/[^\d]/g, ''), 10)
              if (!isNaN(num)) shot.durationMs = num
            } else {
              (shot as any)[key] = value !== '--' ? value : undefined
            }
          }
        }
      }
    }

    // ---- 解析段落 ----
    // 按 ### 分割
    const sections = content.split(/^### /gm)
    for (const section of sections) {
      const firstLineEnd = section.indexOf('\n')
      if (firstLineEnd === -1) continue

      const title = section.substring(0, firstLineEnd).trim()
      const sectionBody = section.substring(firstLineEnd + 1).trim()

      // 跳过空段落
      if (!sectionBody) continue

      // 检查是否为图片类
      const imageKey = IMAGE_SECTION_MAP[title]
      if (imageKey) {
        const imgMatch = sectionBody.match(/!\[.*?\]\((.+?)\)/)
        if (imgMatch) {
          shot[imageKey] = imgMatch[1]
        }
        continue
      }

      // 文本类段落
      const textKey = SECTION_MAP[title]
      if (textKey) {
        // 去除末尾的多余空行和 ---
        const cleanText = sectionBody
          .replace(/\n---\s*$/, '')
          .replace(/\n{2,}$/, '')
          .trim()
        shot[textKey] = cleanText || undefined
      }
    }

    // 只有至少有 shotType 或 dialogue 或 visualDesc 的才认为是有效数据
    if (shot.shotType || shot.dialogue || shot.visualDesc || shot.action) {
      shots.push(shot)
    }
  }

  return shots
}
