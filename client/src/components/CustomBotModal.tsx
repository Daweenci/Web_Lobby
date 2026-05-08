import { useState } from 'react';

const ARCHETYPES = [
  { id: 'chill',     label: 'Chill' },
  { id: 'tryhard',  label: 'Tryhard' },
  { id: 'dad_gamer',label: 'Dad Gamer' },
  { id: 'meme_lord', label: 'Meme Lord' },
  { id: 'quiet',    label: 'Quiet' },
  { id: 'coach',    label: 'Coach' },
  { id: 'tilted',   label: 'Tilted' },
  { id: 'newbie',   label: 'Newbie' },
];

export type CustomBotConfig = {
  archetypeId: string;
  energy: number;
  tilt: number;
  chaos: number;
  humor: number;
  usesEmojis: boolean;
  usesGamingSlang: boolean;
  usesMemes: boolean;
  asksQuestions: boolean;
};

type Props = {
  onClose: () => void;
  onCreate: (config: CustomBotConfig) => void;
};

function Slider({
  label,
  value,
  onChange,
}: {
  label: string;
  value: number;
  onChange: (v: number) => void;
}) {
  return (
    <div className="flex flex-col gap-1">
      <div className="flex justify-between text-xs text-gray-500">
        <span>{label}</span>
        <span className="font-mono font-semibold text-gray-700 w-4 text-right">{value}</span>
      </div>
      <input
        type="range"
        min={1}
        max={5}
        step={1}
        value={value}
        onChange={(e) => onChange(Number(e.target.value))}
        className="w-full accent-purple-500 cursor-pointer h-1.5"
      />
      <div className="flex justify-between text-[10px] text-gray-300 px-0.5">
        <span>1</span><span>2</span><span>3</span><span>4</span><span>5</span>
      </div>
    </div>
  );
}

function Checkbox({
  label,
  value,
  onChange,
}: {
  label: string;
  value: boolean;
  onChange: (v: boolean) => void;
}) {
  return (
    <label className="flex items-center gap-2 text-sm text-gray-700 cursor-pointer select-none">
      <input
        type="checkbox"
        checked={value}
        onChange={(e) => onChange(e.target.checked)}
        className="accent-purple-500 w-4 h-4 cursor-pointer"
      />
      {label}
    </label>
  );
}

export function CustomBotModal({ onClose, onCreate }: Props) {
  const [config, setConfig] = useState<CustomBotConfig>({
    archetypeId: 'chill',
    energy: 3,
    tilt: 2,
    chaos: 2,
    humor: 3,
    usesEmojis: false,
    usesGamingSlang: false,
    usesMemes: false,
    asksQuestions: false,
  });

  const set = <K extends keyof CustomBotConfig>(key: K, val: CustomBotConfig[K]) =>
    setConfig((prev) => ({ ...prev, [key]: val }));

  return (
    <div
      className="fixed inset-0 bg-black/40 flex items-center justify-center z-50"
      onClick={onClose}
    >
      <div
        className="bg-white rounded-2xl shadow-2xl w-full max-w-sm mx-4 p-5 flex flex-col gap-4"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        <div className="flex items-center justify-between">
          <h2 className="text-base font-bold text-gray-800">Create Custom Bot</h2>
          <button
            onClick={onClose}
            className="text-gray-400 hover:text-gray-600 text-xl leading-none"
          >
            ✕
          </button>
        </div>

        {/* Archetype dropdown */}
        <div className="flex flex-col gap-1.5">
          <label className="text-xs font-medium text-gray-500 uppercase tracking-wide">
            Archetype
          </label>
          <select
            value={config.archetypeId}
            onChange={(e) => set('archetypeId', e.target.value)}
            className="w-full border border-gray-200 rounded-xl px-3 py-2 text-sm text-gray-700 focus:outline-none focus:ring-2 focus:ring-purple-400 bg-white"
          >
            {ARCHETYPES.map((a) => (
              <option key={a.id} value={a.id}>
                {a.label}
              </option>
            ))}
          </select>
        </div>

        {/* Sliders */}
        <div className="flex flex-col gap-3">
          <label className="text-xs font-medium text-gray-500 uppercase tracking-wide">
            Personality
          </label>
          <Slider label="⚡ Energy"  value={config.energy} onChange={(v) => set('energy', v)} />
          <Slider label="😤 Tilt"   value={config.tilt}   onChange={(v) => set('tilt', v)} />
          <Slider label="🎲 Chaos"  value={config.chaos}  onChange={(v) => set('chaos', v)} />
          <Slider label="😂 Humor"  value={config.humor}  onChange={(v) => set('humor', v)} />
        </div>

        {/* Checkboxes */}
        <div className="flex flex-col gap-2">
          <label className="text-xs font-medium text-gray-500 uppercase tracking-wide">
            Traits
          </label>
          <Checkbox label="Uses emojis"      value={config.usesEmojis}      onChange={(v) => set('usesEmojis', v)} />
          <Checkbox label="Gaming slang"     value={config.usesGamingSlang} onChange={(v) => set('usesGamingSlang', v)} />
          <Checkbox label="Memes"            value={config.usesMemes}        onChange={(v) => set('usesMemes', v)} />
          <Checkbox label="Asks questions"   value={config.asksQuestions}    onChange={(v) => set('asksQuestions', v)} />
        </div>

        {/* Actions */}
        <div className="flex gap-2 pt-1">
          <button
            onClick={onClose}
            className="flex-1 border border-gray-200 text-gray-600 rounded-xl py-2 text-sm hover:bg-gray-50 transition"
          >
            Cancel
          </button>
          <button
            onClick={() => { onCreate(config); onClose(); }}
            className="flex-1 bg-purple-500 hover:bg-purple-600 text-white rounded-xl py-2 text-sm font-medium transition"
          >
            Create Bot
          </button>
        </div>
      </div>
    </div>
  );
}