const sizes = { sm: "size-6", md: "size-7", lg: "size-10" };

export default function Logo({ size = "md" }: { size?: keyof typeof sizes }) {
  return (
    <span
      aria-hidden="true"
      className={`inline-block shrink-0 bg-accent [mask-image:url('/logo.svg?v=3')] [mask-position:center] [mask-repeat:no-repeat] [mask-size:contain] ${sizes[size]}`}
    />
  );
}
