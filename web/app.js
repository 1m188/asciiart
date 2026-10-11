/**
 * app.js — asciiart 网页版核心逻辑（纯静态，零依赖）
 *
 * 本文件逐模块复刻仓库根目录 Go 版的行为：
 *   chars.go     → RAMP_PRESETS / getRamp / newCustomRamp / mapGrayToChar
 *   image.go     → computeTargetSize / rgbaToGray / scaleImageToImageData
 *   converter.go → grayToAsciiLines / colorAsciiRows / colorAsciiToOutput / convertImage
 *   render.go    → escapeHtml / buildHtmlDocument
 *
 * 纯计算函数不接触 DOM，可在 Node 中直接 require 做单元测试；
 * 浏览器交互集中在文件末尾的 initApp()，仅在有 document 的环境执行。
 */
'use strict';

/* ============================================================
 * 字符梯度（复刻 chars.go）
 * ============================================================ */

// 预设梯度（从暗到亮排列），与 chars.go 的 RampShort/Standard/Detailed/Blocks 逐一对应
var RAMP_PRESETS = {
  short: '@%#*+=-:. ',
  standard: '@%#*+=-:. ',
  detailed: '$@B%8&WM#*oahkbdpqwmZO0QLCJUYXzcvunxrjft/\\|()1{}[]?-_+~<>i!lI;:,"^`\'. ',
  blocks: '█▓▒░ ',
};

// GetRamp：按名称取预设梯度，大小写不敏感；未知名称抛错（错误文案同 Go）
function getRamp(name) {
  var ramp = RAMP_PRESETS[String(name == null ? '' : name).toLowerCase()];
  if (!ramp) {
    var names = Object.keys(RAMP_PRESETS).join(', ');
    throw new Error('未知的字符梯度预设 "' + name + '"，支持的预设: ' + names);
  }
  return Array.from(ramp); // 按 Unicode 码点拆分，等价 Go 的 []rune
}

// NewCustomRamp：从用户字符串创建梯度（从暗到亮）；空字符串抛错（同 Go）
function newCustomRamp(s) {
  if (!s) {
    throw new Error('自定义梯度字符串不能为空');
  }
  return Array.from(s);
}

// MapGrayToChar：index = gray*(n-1)/255（整数截断），invert 时反转索引（同 chars.go）
function mapGrayToChar(gray, ramp, invert) {
  if (!ramp || ramp.length === 0) {
    return ' ';
  }
  var index = Math.trunc((gray * (ramp.length - 1)) / 255);
  if (invert) {
    index = ramp.length - 1 - index;
  }
  return ramp[index];
}

/* ============================================================
 * 预处理参数与灰度化（复刻 image.go）
 * ============================================================ */

// Preprocess 的尺寸部分：目标高度 = 宽 × 宽高比 × 原高 / 原宽（截断），最小 1 行。
// 乘除顺序与 Go 的 int(float64(w)*aspect*ih/iw) 保持一致；
// width<=0 回退 80、aspect<=0 回退 0.5（同 DefaultConvertOptions）。
function computeTargetSize(iw, ih, targetWidth, aspectRatio) {
  var width = targetWidth > 0 ? Math.floor(targetWidth) : 80;
  var aspect = aspectRatio > 0 ? aspectRatio : 0.5;
  var height = Math.trunc(width * aspect * ih / iw);
  if (height < 1) {
    height = 1;
  }
  return { width: width, height: height };
}

// 灰度化。
// 注意：Go 版实际管线是 draw 缩放 → image.Gray.Set → color.GrayModel.Convert，
// 使用定点数并带舍入（r/g/b 为 16 位值）：
//   y = (19595*r + 38470*g + 7471*b + 1<<15) >> 24
// 浏览器 canvas 输出 8 位 sRGB（不透明像素的 16 位值 = 8 位值 × 257），代入展开：
//   gray = ⌊(5035915*R + 9886790*G + 1920047*B + 32768) / 2^24⌋
// 这与 image.go 注释中的浮点公式略有差异，但与 CLI 的真实输出逐位一致。
function rgbaToGray(rgba, pixelCount) {
  var gray = new Uint8Array(pixelCount);
  for (var i = 0; i < pixelCount; i++) {
    var o = i * 4;
    gray[i] = Math.floor(
      (5035915 * rgba[o] + 9886790 * rgba[o + 1] + 1920047 * rgba[o + 2] + 32768) / 16777216
    );
  }
  return gray;
}

/* ============================================================
 * 转换（复刻 converter.go）
 * ============================================================ */

// ConvertFromImage 的字符映射部分：遍历灰度矩阵生成每行字符串
function grayToAsciiLines(gray, width, height, ramp, invert) {
  var lines = new Array(height);
  for (var y = 0; y < height; y++) {
    var row = y * width;
    var line = '';
    for (var x = 0; x < width; x++) {
      line += mapGrayToChar(gray[row + x], ramp, invert);
    }
    lines[y] = line;
  }
  return lines;
}

// ConvertColorFromImage 的结构化版本：每格 {ch, r, g, b}。
// Go 版把颜色编进 ANSI 转义序列；网页版保留结构化数据，最终渲染为 span。
function colorAsciiRows(gray, rgba, width, height, ramp, invert) {
  var rows = new Array(height);
  for (var y = 0; y < height; y++) {
    var row = y * width;
    var out = new Array(width);
    for (var x = 0; x < width; x++) {
      var i = row + x;
      var o = i * 4;
      out[x] = {
        ch: mapGrayToChar(gray[i], ramp, invert),
        r: rgba[o],
        g: rgba[o + 1],
        b: rgba[o + 2]
      };
    }
    rows[y] = out;
  }
  return rows;
}

// 彩色模式输出：plain 为纯字符文本（复制 / 下载 .txt 用），
// html 为逐字符 <span style="color:rgb(...)">（等价 render.go 中 ansiToHTML 的产物）
function colorAsciiToOutput(gray, rgba, width, height, ramp, invert) {
  var rows = colorAsciiRows(gray, rgba, width, height, ramp, invert);
  var plainLines = [];
  var htmlLines = [];
  for (var y = 0; y < rows.length; y++) {
    var plainLine = '';
    var htmlLine = '';
    for (var x = 0; x < rows[y].length; x++) {
      var cell = rows[y][x];
      plainLine += cell.ch;
      htmlLine += '<span style="color:rgb(' + cell.r + ',' + cell.g + ',' + cell.b + ')">' +
        escapeHtml(cell.ch) + '</span>';
    }
    plainLines.push(plainLine);
    htmlLines.push(htmlLine);
  }
  return { plain: plainLines.join('\n'), html: htmlLines.join('\n') };
}

/* ============================================================
 * HTML 渲染（复刻 render.go）
 * ============================================================ */

// html.EscapeString 的等价实现（render.go 用它转义每个字符）。
// 快速路径：绝大多数字符不含敏感字符，避免逐字符正则替换的开销。
function escapeHtml(s) {
  s = String(s);
  if (!/[&<>"']/.test(s)) {
    return s;
  }
  return s.replace(/[&<>"']/g, function (c) {
    return { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c];
  });
}

// RenderToHTML 的等价实现：完整 HTML 文档 + monospace pre + 内联颜色样式。
// 与 Go 版唯一的差异：body 加了黑色背景，浏览器里亮色字符更易观察。
function buildHtmlDocument(bodyHtml, title) {
  return '<!DOCTYPE html>\n<html>\n<head>\n<meta charset="utf-8">\n<title>' + escapeHtml(title) +
    '</title>\n<style>\n' +
    '  body { background: #000; margin: 0; padding: 16px; }\n' +
    '  pre { line-height: 1; font-family: monospace; font-size: 8px; margin: 0; }\n' +
    '  span { display: inline; }\n' +
    '</style>\n</head>\n<body>\n<pre>\n' + bodyHtml + '\n</pre>\n</body>\n</html>\n';
}

/* ============================================================
 * 浏览器转换入口（ConvertFromImage / ConvertColorFromImage 的等价入口）
 * ============================================================ */

// 复用同一块离屏画布：每帧新建 canvas 会带来大量分配与回收开销。
// 重设宽高会自动清空画布并重置 2D 上下文状态，等效于新画布。
var _scratchCanvas = null;
function scaleImageToImageData(img, width, height) {
  if (!_scratchCanvas) {
    _scratchCanvas = document.createElement('canvas');
  }
  _scratchCanvas.width = width;
  _scratchCanvas.height = height;
  var ctx = _scratchCanvas.getContext('2d', { willReadFrequently: true });
  ctx.fillStyle = '#ffffff';
  ctx.fillRect(0, 0, width, height);
  ctx.imageSmoothingEnabled = true;
  ctx.drawImage(img, 0, 0, width, height);
  // getImageData 返回独立副本，此后画布可安全复用
  return ctx.getImageData(0, 0, width, height);
}

// 网页版转换入口：img 为已解码的 HTMLImageElement。
// opts: { width, ramp, customRamp, invert, color, aspectRatio }，对应 CLI 参数。
// 返回结构化数据（gray/rgba），彩色 HTML（span 着色）体积大，
// 只在下载 .html 时按需生成（colorAsciiToOutput），不在每次渲染时构建。
function convertImage(img, opts) {
  var iw = img.naturalWidth || img.width;
  var ih = img.naturalHeight || img.height;
  var size = computeTargetSize(iw, ih, opts.width, opts.aspectRatio);
  var ramp = opts.customRamp ? newCustomRamp(opts.customRamp) : getRamp(opts.ramp);
  var invert = !!opts.invert;

  // 缩放一次：灰度决定字符，同一份像素在彩色模式下兼作颜色来源
  // （与 Go 一致：彩色模式的颜色来自按目标尺寸缩放的原始图）
  var data = scaleImageToImageData(img, size.width, size.height);
  var gray = rgbaToGray(data.data, size.width * size.height);
  var lines = grayToAsciiLines(gray, size.width, size.height, ramp, invert);

  return {
    mode: opts.color ? 'color' : 'gray',
    width: size.width,
    height: size.height,
    plain: lines.join('\n'),
    gray: gray,
    rgba: data.data,
    ramp: ramp,
    invert: invert
  };
}

/* ============================================================
 * 浏览器 UI（以下代码仅在存在 document 的环境中执行）
 * ============================================================ */

function initApp() {
  function $(id) { return document.getElementById(id); }

  var els = {
    dropzone: $('dropzone'),
    file: $('file'),
    preview: $('preview'),
    previewEmpty: $('previewEmpty'),
    widthRange: $('widthRange'),
    widthNum: $('widthNum'),
    ramp: $('ramp'),
    customRampWrap: $('customRampWrap'),
    customRamp: $('customRamp'),
    invert: $('invert'),
    color: $('color'),
    aspect: $('aspect'),
    fontSize: $('fontSize'),
    fontSizeVal: $('fontSizeVal'),
    ascii: $('ascii'),
    asciiCanvas: $('asciiCanvas'),
    info: $('info'),
    error: $('error'),
    placeholder: $('placeholder'),
    copyBtn: $('copyBtn'),
    txtBtn: $('txtBtn'),
    htmlBtn: $('htmlBtn')
  };

  var state = { image: null, url: null, fileName: '', result: null, rafPending: false };

  function showError(msg) {
    els.error.textContent = msg;
    els.error.hidden = !msg;
  }

  function setButtonsEnabled(on) {
    els.copyBtn.disabled = !on;
    els.txtBtn.disabled = !on;
    els.htmlBtn.disabled = !on;
  }

  // ---- 图片加载（对应 LoadImage：文件 → 解码；GIF 天然只取首帧）----

  function loadFile(file) {
    if (!file) return;
    var isImage = (file.type && file.type.indexOf('image/') === 0) ||
      /\.(png|jpe?g|gif|bmp|webp)$/i.test(file.name || '');
    if (!isImage) {
      showError('仅支持 PNG / JPEG / GIF / BMP / WebP 图片：' + file.name);
      return;
    }
    showError('');
    if (state.url) {
      URL.revokeObjectURL(state.url);
    }
    state.url = URL.createObjectURL(file);
    state.fileName = file.name || 'image';

    var img = new Image();
    img.onload = function () {
      state.image = img;
      els.preview.src = state.url;
      els.preview.hidden = false;
      els.previewEmpty.hidden = true;
      render();
    };
    img.onerror = function () {
      state.image = null;
      els.preview.hidden = true;
      els.previewEmpty.hidden = false;
      showError('图片解码失败：' + state.fileName);
    };
    img.src = state.url;
  }

  els.file.addEventListener('change', function () { loadFile(els.file.files[0]); });

  // 拖拽：拖到页面任意位置均可
  window.addEventListener('dragover', function (e) {
    e.preventDefault();
    els.dropzone.classList.add('dragover');
  });
  window.addEventListener('dragleave', function (e) {
    if (!e.relatedTarget) {
      els.dropzone.classList.remove('dragover');
    }
  });
  window.addEventListener('drop', function (e) {
    e.preventDefault();
    els.dropzone.classList.remove('dragover');
    var file = e.dataTransfer && e.dataTransfer.files && e.dataTransfer.files[0];
    if (file) {
      loadFile(file);
    }
  });

  // ---- 选项读取（对应 CLI 参数）----

  function readOptions() {
    return {
      width: parseInt(els.widthNum.value, 10) || 80,
      ramp: els.ramp.value,
      customRamp: els.ramp.value === 'custom' ? els.customRamp.value : '',
      invert: els.invert.checked,
      color: els.color.checked,
      aspectRatio: parseFloat(els.aspect.value) || 0.5
    };
  }

  // ---- 渲染（对应 RenderToWriter / RenderToHTML）----

  function render() {
    if (!state.image) return;

    var t0 = performance.now();
    try {
      state.result = convertImage(state.image, readOptions());
      drawOutput(state.result);
    } catch (err) {
      state.result = null;
      setButtonsEnabled(false);
      showError(err.message);
      return;
    }
    showError('');
    var ms = performance.now() - t0;
    var result = state.result;
    els.info.textContent = result.width + ' × ' + result.height + ' 字符 · ' +
      (result.mode === 'color' ? '彩色' : '灰度') + ' · ' + ms.toFixed(1) + ' ms';
    els.placeholder.hidden = true;
    setButtonsEnabled(true);
  }

  // ---- 输出绘制：灰度走文本节点（可选中、开销小）；
  // ---- 彩色走 canvas——数万字符逐个建 <span> 会让 DOM 与排版开销失控 ----

  var MONO_FONT = 'ui-monospace, "Cascadia Code", Consolas, "Courier New", monospace';
  var COLOR_BG = '#0a0d12'; // 与输出区背景一致

  function currentFontSize() {
    return parseInt(els.fontSize.value, 10) || 10;
  }

  function drawOutput(result) {
    if (result.mode === 'color') {
      els.ascii.hidden = true;
      els.asciiCanvas.hidden = false;
      drawColorCanvas(result);
    } else {
      els.asciiCanvas.hidden = true;
      els.ascii.hidden = false;
      els.ascii.textContent = result.plain;
    }
  }

  // 等宽字符 advance 宽度（与页面同字体栈），用独立小画布测量
  var _measureCtx = null;
  function monoCharWidth(fontSize) {
    if (!_measureCtx) {
      _measureCtx = document.createElement('canvas').getContext('2d');
    }
    _measureCtx.font = fontSize + 'px ' + MONO_FONT;
    return _measureCtx.measureText('M').width;
  }

  function drawColorCanvas(result) {
    var fs = currentFontSize();
    var charW = monoCharWidth(fs);
    var cols = result.width;
    var rows = result.height;
    var cssW = cols * charW;
    var cssH = rows * fs;

    var canvas = els.asciiCanvas;
    canvas.style.width = cssW + 'px';
    canvas.style.height = cssH + 'px';
    // 位图像素上限 16M（≈4096²）：超出时降低有效 DPR，避免极端字号/宽度下位图过大
    var dpr = Math.min(window.devicePixelRatio || 1,
      Math.sqrt(16777216 / Math.max(1, cssW * cssH)));
    if (dpr < 1) dpr = 1;
    canvas.width = Math.max(1, Math.ceil(cssW * dpr));
    canvas.height = Math.max(1, Math.ceil(cssH * dpr));

    var ctx = canvas.getContext('2d');
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
    ctx.font = fs + 'px ' + MONO_FONT;
    ctx.textBaseline = 'top';
    ctx.fillStyle = COLOR_BG;
    ctx.fillRect(0, 0, cssW, cssH);

    var gray = result.gray;
    var rgba = result.rgba;
    var ramp = result.ramp;
    var invert = result.invert;
    var lastColor = '';
    for (var y = 0; y < rows; y++) {
      var row = y * cols;
      var py = y * fs;
      for (var x = 0; x < cols; x++) {
        var i = row + x;
        var ch = mapGrayToChar(gray[i], ramp, invert);
        if (ch === ' ') {
          continue; // 空白格即背景色，无需绘制
        }
        var o = i * 4;
        var color = 'rgb(' + rgba[o] + ',' + rgba[o + 1] + ',' + rgba[o + 2] + ')';
        if (color !== lastColor) {
          ctx.fillStyle = color; // 相邻同色（平坦区域）跳过重复设置
          lastColor = color;
        }
        ctx.fillText(ch, x * charW, py);
      }
    }
  }

  // 滑块连续拖动时节流：每帧最多重算一次
  function scheduleRender() {
    if (state.rafPending) return;
    state.rafPending = true;
    requestAnimationFrame(function () {
      state.rafPending = false;
      render();
    });
  }

  // ---- 控件事件 ----

  els.widthRange.addEventListener('input', function () {
    els.widthNum.value = els.widthRange.value;
    scheduleRender();
  });
  els.widthNum.addEventListener('input', function () {
    var v = parseInt(els.widthNum.value, 10);
    if (!isNaN(v)) {
      if (v < 10) v = 10;
      if (v > 500) v = 500;
      els.widthRange.value = String(Math.min(Math.max(v, 20), 300));
    }
    scheduleRender();
  });
  els.ramp.addEventListener('change', function () {
    els.customRampWrap.hidden = els.ramp.value !== 'custom';
    scheduleRender();
  });
  els.customRamp.addEventListener('input', scheduleRender);
  els.invert.addEventListener('change', scheduleRender);
  els.color.addEventListener('change', scheduleRender);
  els.aspect.addEventListener('change', function () {
    var v = parseFloat(els.aspect.value);
    if (isNaN(v) || v <= 0) v = 0.5;
    if (v < 0.05) v = 0.05;
    if (v > 2) v = 2;
    els.aspect.value = String(v);
    scheduleRender();
  });
  els.fontSize.addEventListener('input', function () {
    var fs = currentFontSize();
    els.fontSizeVal.textContent = fs + 'px';
    if (state.result && state.result.mode === 'color') {
      drawColorCanvas(state.result); // 仅重绘画布，无需重新转换
    } else {
      els.ascii.style.fontSize = fs + 'px';
    }
  });

  // ---- 导出 ----

  function download(content, mime, ext) {
    var base = (state.fileName || 'image').replace(/\.[^.]+$/, '');
    var blob = new Blob([content], { type: mime });
    var url = URL.createObjectURL(blob);
    var a = document.createElement('a');
    a.href = url;
    a.download = 'asciiart-' + base + ext;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    setTimeout(function () { URL.revokeObjectURL(url); }, 1000);
  }

  function flashButton(btn, text) {
    var old = btn.textContent;
    btn.textContent = text;
    setTimeout(function () { btn.textContent = old; }, 1200);
  }

  function fallbackCopy(text) {
    var ta = document.createElement('textarea');
    ta.value = text;
    ta.style.position = 'fixed';
    ta.style.opacity = '0';
    document.body.appendChild(ta);
    ta.select();
    try { document.execCommand('copy'); } catch (e) { /* 忽略：无剪贴板权限时静默失败 */ }
    document.body.removeChild(ta);
  }

  els.copyBtn.addEventListener('click', function () {
    if (!state.result) return;
    var text = state.result.plain;
    if (navigator.clipboard && navigator.clipboard.writeText) {
      navigator.clipboard.writeText(text).then(
        function () { flashButton(els.copyBtn, '已复制 ✓'); },
        function () { fallbackCopy(text); flashButton(els.copyBtn, '已复制 ✓'); }
      );
    } else {
      fallbackCopy(text);
      flashButton(els.copyBtn, '已复制 ✓');
    }
  });

  els.txtBtn.addEventListener('click', function () {
    if (state.result) {
      download(state.result.plain, 'text/plain;charset=utf-8', '.txt');
    }
  });

  els.htmlBtn.addEventListener('click', function () {
    if (!state.result) return;
    var r = state.result;
    // span 着色 HTML 按需生成（对应 render.go RenderToHTML）
    var bodyHtml = r.mode === 'color'
      ? colorAsciiToOutput(r.gray, r.rgba, r.width, r.height, r.ramp, r.invert).html
      : escapeHtml(r.plain);
    download(
      buildHtmlDocument(bodyHtml, 'ASCII Art - ' + state.fileName),
      'text/html;charset=utf-8',
      '.html'
    );
  });

  setButtonsEnabled(false);
}

/* ---- 环境启动 ---- */

// 供 Node 单元测试 require（浏览器中 module 为 undefined，自动忽略）
if (typeof module !== 'undefined' && module.exports) {
  module.exports = {
    RAMP_PRESETS: RAMP_PRESETS,
    getRamp: getRamp,
    newCustomRamp: newCustomRamp,
    mapGrayToChar: mapGrayToChar,
    computeTargetSize: computeTargetSize,
    rgbaToGray: rgbaToGray,
    grayToAsciiLines: grayToAsciiLines,
    colorAsciiRows: colorAsciiRows,
    colorAsciiToOutput: colorAsciiToOutput,
    escapeHtml: escapeHtml,
    buildHtmlDocument: buildHtmlDocument
  };
}

// 仅在浏览器中启动 UI
if (typeof document !== 'undefined') {
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', initApp);
  } else {
    initApp();
  }
}
