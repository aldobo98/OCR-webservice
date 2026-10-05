//import { Carousel, CarouselContent, CarouselItem, CarouselPrevious, CarouselNext } from './components/ui/carousel'
//import nobrain from '../public/ nobrain.jpg'
//import money from '../public/money.jpg'
//import { useState } from 'react';
import ImageForm from './components/ui/image-form'
//import ImageCard from './components/ui/image-card';

export function App() {
/*
  const [img, setImg] = useState<{image: File | null, text: string}>({
    image: null,
    text: "",
  })
  */

  //const [processing, setProcessing] = useState(false);

  return (
    <div className="flex place-content-center gap-5">
      <div>
        <title>Karakterfelismerés</title>
        <h1 className="text-2xl pb-8 pt-8">Karakterfelismerés EasyOCR használatával</h1>
        <ImageForm></ImageForm>
      
      </div>
    </div>
  )
}

/*
{processing && (
          <div className="flex flex-col items-center justify-center gap-4">
            <div className="h-10 w-10 animate-spin rounded-full border-4 border-black border-t-transparent" />
            <p>Kérjük, várjon...</p>
          </div>
        )}

        {img.image != null && (
          <div>
            <ImageCard imageUrl={URL.createObjectURL(img.image)} caption={img.text}></ImageCard>
          </div>
        )}
*/
export default App
