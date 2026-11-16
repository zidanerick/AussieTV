// Command iconbuilder creates PNG and ICO images using only the Go standard library.
// Shape geometry mirrors static/app-icon.svg; no external font/image services needed.
package main

import (
 "bytes"
 "encoding/binary"
 "fmt"
 "image"
 "image/color"
 "image/png"
 "math"
 "os"
 "path/filepath"
)

type point struct { x,y float64 }
func line(x,y float64,a,b point,r float64)bool{
 dx,dy:=b.x-a.x,b.y-a.y;t:=((x-a.x)*dx+(y-a.y)*dy)/(dx*dx+dy*dy)
 t=math.Max(0,math.Min(1,t));xx,yy:=x-(a.x+t*dx),y-(a.y+t*dy)
 return xx*xx+yy*yy<=r*r
}
func roundrect(x,y float64)bool{
 if x<9||x>119||y<35||y>115{return false}
 cx:=math.Max(28,math.Min(100,x));cy:=math.Max(54,math.Min(96,y))
 return math.Hypot(x-cx,y-cy)<=19
}
func triangle(x,y float64)bool{
 a,b,c:=point{43,50},point{94,78},point{43,106}
 cross:=func(p,q point)float64{return p.x*q.y-p.y*q.x}
 p:=point{x,y}
 f:=cross(point{b.x-a.x,b.y-a.y},point{p.x-a.x,p.y-a.y})
 g:=cross(point{c.x-b.x,c.y-b.y},point{p.x-b.x,p.y-b.y})
 h:=cross(point{a.x-c.x,a.y-c.y},point{p.x-c.x,p.y-c.y})
 return(f>=0&&g>=0&&h>=0)||(f<=0&&g<=0&&h<=0)
}
func sample(x,y float64)color.RGBA{
 navy:=color.RGBA{16,41,101,255};aqua:=color.RGBA{20,226,199,255};gold:=color.RGBA{255,203,64,255}
 c:=color.RGBA{}
 if line(x,y,point{36,109},point{30,119},6)||line(x,y,point{92,109},point{98,119},6){c=navy}
 if line(x,y,point{56,40},point{35,18},5)||line(x,y,point{72,40},point{93,18},5){c=navy}
 if math.Hypot(x-35,y-18)<=8||math.Hypot(x-93,y-18)<=8{c=navy}
 if roundrect(x,y){c=navy
 if x>68&&x<111&&y>38&&y<66{
 high:=48.-9*math.Sin((x-68)/43*math.Pi*.75)
 low:=50.+16*math.Pow(math.Max(0,(x-68)/43),2)
 if y>=high&&y<=low{c=gold}
 }
 if triangle(x,y){c=aqua}
 }
 return c
}
func raster(size int)*image.RGBA{
 im:=image.NewRGBA(image.Rect(0,0,size,size))
 for y:=0;y<size;y++{for x:=0;x<size;x++{
 var rr,gg,bb,aa int
 for sy:=0;sy<3;sy++{for sx:=0;sx<3;sx++{
 p:=sample((float64(x)+(float64(sx)+.5)/3)*128/float64(size),(float64(y)+(float64(sy)+.5)/3)*128/float64(size))
 rr+=int(p.R)*int(p.A);gg+=int(p.G)*int(p.A);bb+=int(p.B)*int(p.A);aa+=int(p.A)
 }}
 if aa>0{im.SetRGBA(x,y,color.RGBA{uint8(rr/aa),uint8(gg/aa),uint8(bb/aa),uint8(aa/9)})}
 }}
 return im
}
func encode(im image.Image)([]byte,error){var b bytes.Buffer;err:=png.Encode(&b,im);return b.Bytes(),err}
func main(){
 if err:=os.MkdirAll("assets",0755);err!=nil{panic(err)}
 if err:=os.MkdirAll("static",0755);err!=nil{panic(err)}
 for _,size:=range []int{32,192}{
 data,err:=encode(raster(size));if err!=nil{panic(err)}
 if err=os.WriteFile(filepath.Join("static",fmt.Sprintf("favicon-%d.png",size)),data,0644);err!=nil{panic(err)}
 }
 sizes:=[]int{16,24,32,48,64,128,256};imgs:=make([][]byte,len(sizes))
 for i,s:=range sizes{data,err:=encode(raster(s));if err!=nil{panic(err)};imgs[i]=data}
 var ico bytes.Buffer
 _=binary.Write(&ico,binary.LittleEndian,uint16(0))
 _=binary.Write(&ico,binary.LittleEndian,uint16(1))
 _=binary.Write(&ico,binary.LittleEndian,uint16(len(sizes)))
 offset:=6+16*len(sizes)
 for i,s:=range sizes{
 side:=byte(s);if s==256{side=0}
 _,_=ico.Write([]byte{side,side,0,0})
 _=binary.Write(&ico,binary.LittleEndian,uint16(1))
 _=binary.Write(&ico,binary.LittleEndian,uint16(32))
 _=binary.Write(&ico,binary.LittleEndian,uint32(len(imgs[i])))
 _=binary.Write(&ico,binary.LittleEndian,uint32(offset))
 offset+=len(imgs[i])
 }
 for _,data:=range imgs{_,_=ico.Write(data)}
 for _,path:=range []string{"assets/AussieTV.ico","static/favicon.ico"}{if err:=os.WriteFile(path,ico.Bytes(),0644);err!=nil{panic(err)}}
 fmt.Println("Generated AussieTV brand icons (PNG and ICO)")
}
