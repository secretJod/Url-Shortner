import { clsx } from 'clsx';
import { motion } from 'framer-motion';

export default function Spinner({ size = 'md', className = '' }) {
  const sizes = {
    sm: 'w-4 h-4',
    md: 'w-8 h-8',
    lg: 'w-12 h-12'
  };

  return (
    <div className={clsx("flex justify-center items-center p-4", className)}>
      <div className="relative">
        {/* Pulsing glow halo */}
        <motion.div
          className={clsx("absolute inset-0 rounded-full blur-md", sizes[size])}
          style={{
            background: 'conic-gradient(from 0deg, #0ea5e9, #8b5cf6, #0ea5e9)',
          }}
          animate={{ rotate: 360, scale: [1, 1.1, 1] }}
          transition={{
            rotate: { duration: 1, repeat: Infinity, ease: 'linear' },
            scale: { duration: 1.5, repeat: Infinity, ease: 'easeInOut' },
          }}
        />
        {/* Spinning gradient ring */}
        <motion.div
          className={clsx(
            "relative rounded-full border-2 border-transparent",
            sizes[size]
          )}
          style={{
            background: 'linear-gradient(135deg, #0ea5e9, #8b5cf6) border-box',
            WebkitMask:
              'linear-gradient(#fff 0 0) padding-box, linear-gradient(#fff 0 0)',
            WebkitMaskComposite: 'xor',
            maskComposite: 'exclude',
          }}
          animate={{ rotate: 360 }}
          transition={{ duration: 0.8, repeat: Infinity, ease: 'linear' }}
        />
      </div>
    </div>
  );
}
