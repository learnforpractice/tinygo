; ModuleID = 'slice.go'
source_filename = "slice.go"
target datalayout = "e-m:e-p:32:32-p10:8:8-p20:8:8-i64:64-n32:64-S128-ni:1:10:20"
target triple = "wasm32-unknown-wasi"

declare noalias nonnull i8* @runtime.alloc(i32, i8*, i8*) #0

declare void @runtime.trackPointer(i8* nocapture readonly, i8*) #0

; Function Attrs: nounwind
define hidden void @main.init(i8* %context) unnamed_addr #1 {
entry:
  ret void
}

; Function Attrs: nounwind
define hidden i32 @main.sliceLen(i32* %ints.data, i32 %ints.len, i32 %ints.cap, i8* %context) unnamed_addr #1 {
entry:
  ret i32 %ints.len
}

; Function Attrs: nounwind
define hidden i32 @main.sliceCap(i32* %ints.data, i32 %ints.len, i32 %ints.cap, i8* %context) unnamed_addr #1 {
entry:
  ret i32 %ints.cap
}

; Function Attrs: nounwind
define hidden i32 @main.sliceElement(i32* %ints.data, i32 %ints.len, i32 %ints.cap, i32 %index, i8* %context) unnamed_addr #1 {
entry:
  %.not = icmp ult i32 %index, %ints.len
  br i1 %.not, label %lookup.next, label %lookup.throw

lookup.next:                                      ; preds = %entry
  %0 = getelementptr inbounds i32, i32* %ints.data, i32 %index
  %1 = load i32, i32* %0, align 4
  ret i32 %1

lookup.throw:                                     ; preds = %entry
  call void @runtime.lookupPanic(i8* undef) #2
  unreachable
}

declare void @runtime.lookupPanic(i8*) #0

; Function Attrs: nounwind
define hidden { i32*, i32, i32 } @main.sliceAppendValues(i32* %ints.data, i32 %ints.len, i32 %ints.cap, i8* %context) unnamed_addr #1 {
entry:
  %varargs = call i8* @runtime.alloc(i32 12, i8* nonnull inttoptr (i32 3 to i8*), i8* undef) #2
  call void @runtime.trackPointer(i8* nonnull %varargs, i8* undef) #2
  %0 = bitcast i8* %varargs to i32*
  store i32 1, i32* %0, align 4
  %1 = getelementptr inbounds i8, i8* %varargs, i32 4
  %2 = bitcast i8* %1 to i32*
  store i32 2, i32* %2, align 4
  %3 = getelementptr inbounds i8, i8* %varargs, i32 8
  %4 = bitcast i8* %3 to i32*
  store i32 3, i32* %4, align 4
  %append.srcPtr = bitcast i32* %ints.data to i8*
  %append.new = call { i8*, i32, i32 } @runtime.sliceAppend(i8* %append.srcPtr, i8* nonnull %varargs, i32 %ints.len, i32 %ints.cap, i32 3, i32 4, i8* undef) #2
  %append.newPtr = extractvalue { i8*, i32, i32 } %append.new, 0
  %append.newBuf = bitcast i8* %append.newPtr to i32*
  %append.newLen = extractvalue { i8*, i32, i32 } %append.new, 1
  %append.newCap = extractvalue { i8*, i32, i32 } %append.new, 2
  %5 = insertvalue { i32*, i32, i32 } undef, i32* %append.newBuf, 0
  %6 = insertvalue { i32*, i32, i32 } %5, i32 %append.newLen, 1
  %7 = insertvalue { i32*, i32, i32 } %6, i32 %append.newCap, 2
  call void @runtime.trackPointer(i8* %append.newPtr, i8* undef) #2
  ret { i32*, i32, i32 } %7
}

declare { i8*, i32, i32 } @runtime.sliceAppend(i8*, i8* nocapture readonly, i32, i32, i32, i32, i8*) #0

; Function Attrs: nounwind
define hidden { i32*, i32, i32 } @main.sliceAppendSlice(i32* %ints.data, i32 %ints.len, i32 %ints.cap, i32* %added.data, i32 %added.len, i32 %added.cap, i8* %context) unnamed_addr #1 {
entry:
  %append.srcPtr = bitcast i32* %ints.data to i8*
  %append.srcPtr1 = bitcast i32* %added.data to i8*
  %append.new = call { i8*, i32, i32 } @runtime.sliceAppend(i8* %append.srcPtr, i8* %append.srcPtr1, i32 %ints.len, i32 %ints.cap, i32 %added.len, i32 4, i8* undef) #2
  %append.newPtr = extractvalue { i8*, i32, i32 } %append.new, 0
  %append.newBuf = bitcast i8* %append.newPtr to i32*
  %append.newLen = extractvalue { i8*, i32, i32 } %append.new, 1
  %append.newCap = extractvalue { i8*, i32, i32 } %append.new, 2
  %0 = insertvalue { i32*, i32, i32 } undef, i32* %append.newBuf, 0
  %1 = insertvalue { i32*, i32, i32 } %0, i32 %append.newLen, 1
  %2 = insertvalue { i32*, i32, i32 } %1, i32 %append.newCap, 2
  call void @runtime.trackPointer(i8* %append.newPtr, i8* undef) #2
  ret { i32*, i32, i32 } %2
}

; Function Attrs: nounwind
define hidden i32 @main.sliceCopy(i32* %dst.data, i32 %dst.len, i32 %dst.cap, i32* %src.data, i32 %src.len, i32 %src.cap, i8* %context) unnamed_addr #1 {
entry:
  %copy.dstPtr = bitcast i32* %dst.data to i8*
  %copy.srcPtr = bitcast i32* %src.data to i8*
  %copy.n = call i32 @runtime.sliceCopy(i8* %copy.dstPtr, i8* %copy.srcPtr, i32 %dst.len, i32 %src.len, i32 4, i8* undef) #2
  ret i32 %copy.n
}

declare i32 @runtime.sliceCopy(i8* nocapture writeonly, i8* nocapture readonly, i32, i32, i32, i8*) #0

; Function Attrs: nounwind
define hidden { i8*, i32, i32 } @main.makeByteSlice(i32 %len, i8* %context) unnamed_addr #1 {
entry:
  %slice.maxcap = icmp slt i32 %len, 0
  br i1 %slice.maxcap, label %slice.throw, label %slice.next

slice.next:                                       ; preds = %entry
  %makeslice.buf = call i8* @runtime.alloc(i32 %len, i8* nonnull inttoptr (i32 3 to i8*), i8* undef) #2
  %0 = insertvalue { i8*, i32, i32 } undef, i8* %makeslice.buf, 0
  %1 = insertvalue { i8*, i32, i32 } %0, i32 %len, 1
  %2 = insertvalue { i8*, i32, i32 } %1, i32 %len, 2
  call void @runtime.trackPointer(i8* nonnull %makeslice.buf, i8* undef) #2
  ret { i8*, i32, i32 } %2

slice.throw:                                      ; preds = %entry
  call void @runtime.slicePanic(i8* undef) #2
  unreachable
}

declare void @runtime.slicePanic(i8*) #0

; Function Attrs: nounwind
define hidden { i16*, i32, i32 } @main.makeInt16Slice(i32 %len, i8* %context) unnamed_addr #1 {
entry:
  %slice.maxcap = icmp slt i32 %len, 0
  br i1 %slice.maxcap, label %slice.throw, label %slice.next

slice.next:                                       ; preds = %entry
  %makeslice.cap = shl i32 %len, 1
  %makeslice.buf = call i8* @runtime.alloc(i32 %makeslice.cap, i8* nonnull inttoptr (i32 3 to i8*), i8* undef) #2
  %makeslice.array = bitcast i8* %makeslice.buf to i16*
  %0 = insertvalue { i16*, i32, i32 } undef, i16* %makeslice.array, 0
  %1 = insertvalue { i16*, i32, i32 } %0, i32 %len, 1
  %2 = insertvalue { i16*, i32, i32 } %1, i32 %len, 2
  call void @runtime.trackPointer(i8* nonnull %makeslice.buf, i8* undef) #2
  ret { i16*, i32, i32 } %2

slice.throw:                                      ; preds = %entry
  call void @runtime.slicePanic(i8* undef) #2
  unreachable
}

; Function Attrs: nounwind
define hidden { [3 x i8]*, i32, i32 } @main.makeArraySlice(i32 %len, i8* %context) unnamed_addr #1 {
entry:
  %slice.maxcap = icmp ugt i32 %len, 1431655765
  br i1 %slice.maxcap, label %slice.throw, label %slice.next

slice.next:                                       ; preds = %entry
  %makeslice.cap = mul i32 %len, 3
  %makeslice.buf = call i8* @runtime.alloc(i32 %makeslice.cap, i8* nonnull inttoptr (i32 3 to i8*), i8* undef) #2
  %makeslice.array = bitcast i8* %makeslice.buf to [3 x i8]*
  %0 = insertvalue { [3 x i8]*, i32, i32 } undef, [3 x i8]* %makeslice.array, 0
  %1 = insertvalue { [3 x i8]*, i32, i32 } %0, i32 %len, 1
  %2 = insertvalue { [3 x i8]*, i32, i32 } %1, i32 %len, 2
  call void @runtime.trackPointer(i8* nonnull %makeslice.buf, i8* undef) #2
  ret { [3 x i8]*, i32, i32 } %2

slice.throw:                                      ; preds = %entry
  call void @runtime.slicePanic(i8* undef) #2
  unreachable
}

; Function Attrs: nounwind
define hidden { i32*, i32, i32 } @main.makeInt32Slice(i32 %len, i8* %context) unnamed_addr #1 {
entry:
  %slice.maxcap = icmp ugt i32 %len, 1073741823
  br i1 %slice.maxcap, label %slice.throw, label %slice.next

slice.next:                                       ; preds = %entry
  %makeslice.cap = shl i32 %len, 2
  %makeslice.buf = call i8* @runtime.alloc(i32 %makeslice.cap, i8* nonnull inttoptr (i32 3 to i8*), i8* undef) #2
  %makeslice.array = bitcast i8* %makeslice.buf to i32*
  %0 = insertvalue { i32*, i32, i32 } undef, i32* %makeslice.array, 0
  %1 = insertvalue { i32*, i32, i32 } %0, i32 %len, 1
  %2 = insertvalue { i32*, i32, i32 } %1, i32 %len, 2
  call void @runtime.trackPointer(i8* nonnull %makeslice.buf, i8* undef) #2
  ret { i32*, i32, i32 } %2

slice.throw:                                      ; preds = %entry
  call void @runtime.slicePanic(i8* undef) #2
  unreachable
}

; Function Attrs: nounwind
define hidden i8* @main.Add32(i8* %p, i32 %len, i8* %context) unnamed_addr #1 {
entry:
  %0 = getelementptr i8, i8* %p, i32 %len
  call void @runtime.trackPointer(i8* %0, i8* undef) #2
  ret i8* %0
}

; Function Attrs: nounwind
define hidden i8* @main.Add64(i8* %p, i64 %len, i8* %context) unnamed_addr #1 {
entry:
  %0 = trunc i64 %len to i32
  %1 = getelementptr i8, i8* %p, i32 %0
  call void @runtime.trackPointer(i8* %1, i8* undef) #2
  ret i8* %1
}

; Function Attrs: nounwind
define hidden [4 x i32]* @main.SliceToArray(i32* %s.data, i32 %s.len, i32 %s.cap, i8* %context) unnamed_addr #1 {
entry:
  %0 = icmp ult i32 %s.len, 4
  br i1 %0, label %slicetoarray.throw, label %slicetoarray.next

slicetoarray.next:                                ; preds = %entry
  %1 = bitcast i32* %s.data to [4 x i32]*
  ret [4 x i32]* %1

slicetoarray.throw:                               ; preds = %entry
  call void @runtime.sliceToArrayPointerPanic(i8* undef) #2
  unreachable
}

declare void @runtime.sliceToArrayPointerPanic(i8*) #0

; Function Attrs: nounwind
define hidden [4 x i32]* @main.SliceToArrayConst(i8* %context) unnamed_addr #1 {
entry:
  %makeslice = call i8* @runtime.alloc(i32 24, i8* nonnull inttoptr (i32 3 to i8*), i8* undef) #2
  call void @runtime.trackPointer(i8* nonnull %makeslice, i8* undef) #2
  br i1 false, label %slicetoarray.throw, label %slicetoarray.next

slicetoarray.next:                                ; preds = %entry
  %0 = bitcast i8* %makeslice to [4 x i32]*
  ret [4 x i32]* %0

slicetoarray.throw:                               ; preds = %entry
  unreachable
}

; Function Attrs: nounwind
define hidden { i32*, i32, i32 } @main.SliceInt(i32* dereferenceable_or_null(4) %ptr, i32 %len, i8* %context) unnamed_addr #1 {
entry:
  %0 = icmp ugt i32 %len, 1073741823
  %1 = icmp eq i32* %ptr, null
  %2 = icmp ne i32 %len, 0
  %3 = and i1 %1, %2
  %4 = or i1 %3, %0
  br i1 %4, label %unsafe.Slice.throw, label %unsafe.Slice.next

unsafe.Slice.next:                                ; preds = %entry
  %5 = insertvalue { i32*, i32, i32 } undef, i32* %ptr, 0
  %6 = insertvalue { i32*, i32, i32 } %5, i32 %len, 1
  %7 = insertvalue { i32*, i32, i32 } %6, i32 %len, 2
  %8 = bitcast i32* %ptr to i8*
  call void @runtime.trackPointer(i8* %8, i8* undef) #2
  ret { i32*, i32, i32 } %7

unsafe.Slice.throw:                               ; preds = %entry
  call void @runtime.unsafeSlicePanic(i8* undef) #2
  unreachable
}

declare void @runtime.unsafeSlicePanic(i8*) #0

; Function Attrs: nounwind
define hidden { i8*, i32, i32 } @main.SliceUint16(i8* dereferenceable_or_null(1) %ptr, i16 %len, i8* %context) unnamed_addr #1 {
entry:
  %0 = icmp eq i8* %ptr, null
  %1 = icmp ne i16 %len, 0
  %2 = and i1 %0, %1
  br i1 %2, label %unsafe.Slice.throw, label %unsafe.Slice.next

unsafe.Slice.next:                                ; preds = %entry
  %3 = zext i16 %len to i32
  %4 = insertvalue { i8*, i32, i32 } undef, i8* %ptr, 0
  %5 = insertvalue { i8*, i32, i32 } %4, i32 %3, 1
  %6 = insertvalue { i8*, i32, i32 } %5, i32 %3, 2
  call void @runtime.trackPointer(i8* %ptr, i8* undef) #2
  ret { i8*, i32, i32 } %6

unsafe.Slice.throw:                               ; preds = %entry
  call void @runtime.unsafeSlicePanic(i8* undef) #2
  unreachable
}

; Function Attrs: nounwind
define hidden { i32*, i32, i32 } @main.SliceUint64(i32* dereferenceable_or_null(4) %ptr, i64 %len, i8* %context) unnamed_addr #1 {
entry:
  %0 = icmp ugt i64 %len, 1073741823
  %1 = icmp eq i32* %ptr, null
  %2 = icmp ne i64 %len, 0
  %3 = and i1 %1, %2
  %4 = or i1 %3, %0
  br i1 %4, label %unsafe.Slice.throw, label %unsafe.Slice.next

unsafe.Slice.next:                                ; preds = %entry
  %5 = trunc i64 %len to i32
  %6 = insertvalue { i32*, i32, i32 } undef, i32* %ptr, 0
  %7 = insertvalue { i32*, i32, i32 } %6, i32 %5, 1
  %8 = insertvalue { i32*, i32, i32 } %7, i32 %5, 2
  %9 = bitcast i32* %ptr to i8*
  call void @runtime.trackPointer(i8* %9, i8* undef) #2
  ret { i32*, i32, i32 } %8

unsafe.Slice.throw:                               ; preds = %entry
  call void @runtime.unsafeSlicePanic(i8* undef) #2
  unreachable
}

; Function Attrs: nounwind
define hidden { i32*, i32, i32 } @main.SliceInt64(i32* dereferenceable_or_null(4) %ptr, i64 %len, i8* %context) unnamed_addr #1 {
entry:
  %0 = icmp ugt i64 %len, 1073741823
  %1 = icmp eq i32* %ptr, null
  %2 = icmp ne i64 %len, 0
  %3 = and i1 %1, %2
  %4 = or i1 %3, %0
  br i1 %4, label %unsafe.Slice.throw, label %unsafe.Slice.next

unsafe.Slice.next:                                ; preds = %entry
  %5 = trunc i64 %len to i32
  %6 = insertvalue { i32*, i32, i32 } undef, i32* %ptr, 0
  %7 = insertvalue { i32*, i32, i32 } %6, i32 %5, 1
  %8 = insertvalue { i32*, i32, i32 } %7, i32 %5, 2
  %9 = bitcast i32* %ptr to i8*
  call void @runtime.trackPointer(i8* %9, i8* undef) #2
  ret { i32*, i32, i32 } %8

unsafe.Slice.throw:                               ; preds = %entry
  call void @runtime.unsafeSlicePanic(i8* undef) #2
  unreachable
}

attributes #0 = { "target-features"="+bulk-memory,+nontrapping-fptoint,+sign-ext" }
attributes #1 = { nounwind "target-features"="+bulk-memory,+nontrapping-fptoint,+sign-ext" }
attributes #2 = { nounwind }
