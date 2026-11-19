import { cn } from "@/lib/utils"

type Props = {
  imageUrl: string
  caption?: string
  className?: string
}

export default function ImageCard({ imageUrl, caption, className }: Props) {
  return (
    <figure
      className={cn(
        "w-[500px] overflow-hidden rounded-base border-2 border-border bg-main font-base shadow-shadow",
        className,
      )}
    >
      {/* A kép megtartja a saját képarányát; nagyon magas képeknél a magasság korlátozva, torzítás nélkül */}
      <img className="block w-full h-auto max-h-[75vh] object-contain" src={imageUrl} alt="image" />
      {caption !== undefined && (
        <figcaption className="border-t-2 text-main-foreground border-border p-4 whitespace-pre-wrap">
          {caption}
        </figcaption>
      )}
    </figure>
  )
}
