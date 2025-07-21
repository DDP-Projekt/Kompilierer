use std::ffi::c_void;

use ddpruntime::{
    ddptypes::{DDPAny, DDPChar, DDPInt, DDPList, DDPString, DDPVTable},
    runtime::ddp_panic,
};

fn claim_non_primitive(vtable: &DDPVTable, elem: *mut c_void, any: &DDPAny) {
    if vtable.deep_copy_func.is_some() {
        unsafe {
            vtable.deep_copy_func.unwrap()(
                elem,
                if any.is_standard_value() {
                    std::ptr::from_ref(any) as *mut c_void
                } else {
                    any.value_ptr()
                },
            );
        }
    }
}

#[unsafe(no_mangle)]
pub extern "C" fn efficient_list_append(
    list: &mut DDPList<c_void>,
    elem: *mut c_void,
    any: &DDPAny,
) {
    let vtable = any.get_generic_vtable();

    list.grow_if_needed(vtable.type_size, 1);
    unsafe {
        std::ptr::copy_nonoverlapping(
            elem,
            list.arr.add((list.len * vtable.type_size) as usize),
            vtable.type_size as usize,
        );
    }
    list.len += 1;

    claim_non_primitive(vtable, elem, any);
}

#[unsafe(no_mangle)]
pub extern "C" fn efficient_list_prepend(
    list: &mut DDPList<c_void>,
    elem: *mut c_void,
    any: &DDPAny,
) {
    let vtable = any.get_generic_vtable();

    list.grow_if_needed(vtable.type_size, 1);
    unsafe {
        std::ptr::copy(
            list.arr,
            list.arr.add(vtable.type_size as usize),
            (list.len * vtable.type_size) as usize,
        );
        std::ptr::copy_nonoverlapping(elem, list.arr, vtable.type_size as usize);
    }
    list.len += 1;

    claim_non_primitive(vtable, elem, any);
}

#[unsafe(no_mangle)]
pub extern "C" fn efficient_list_append_list(
    list: &mut DDPList<c_void>,
    other: &mut DDPList<c_void>,
    any: &DDPAny,
) {
    let vtable = any.get_generic_vtable();

    list.grow_if_needed(vtable.type_size, other.len);
    unsafe {
        std::ptr::copy_nonoverlapping(
            other.arr,
            list.arr.add((list.len * vtable.type_size) as usize),
            (vtable.type_size * other.len) as usize,
        );
    }
    list.len += other.len;

    for i in 0..other.len {
        claim_non_primitive(
            vtable,
            unsafe { other.arr.add((i * vtable.type_size) as usize) },
            any,
        );
    }
}

#[unsafe(no_mangle)]
pub extern "C" fn efficient_list_prepend_list(
    list: &mut DDPList<c_void>,
    other: &mut DDPList<c_void>,
    any: &DDPAny,
) {
    let vtable = any.get_generic_vtable();

    list.grow_if_needed(vtable.type_size, other.len);
    unsafe {
        std::ptr::copy(
            list.arr,
            list.arr.add((other.len * vtable.type_size) as usize),
            (list.len * vtable.type_size) as usize,
        );
        std::ptr::copy_nonoverlapping(other.arr, list.arr, (other.len * vtable.type_size) as usize);
    }
    list.len += other.len;

    for i in 0..other.len {
        claim_non_primitive(
            vtable,
            unsafe { other.arr.add((i * vtable.type_size) as usize) },
            any,
        );
    }
}

#[unsafe(no_mangle)]
pub extern "C" fn efficient_list_delete_range(
    list: &mut DDPList<c_void>,
    start: DDPInt,
    end: DDPInt,
    any: &DDPAny,
) {
    if list.len < 0 {
        return;
    }

    if start > end {
        ddp_panic(
            1,
            format!("start index ist größer als end index ({start}, {end})"),
        );
    }

    let vtable = any.get_generic_vtable();

    let start = start.clamp(start, list.len);
    let end = start.clamp(end, list.len);

    if vtable.free_func.is_some() {
        for i in start..=end {
            unsafe {
                vtable.free_func.unwrap()(list.arr.add((i * vtable.type_size) as usize));
            }
        }
    }

    let new_len = list.len - (end - start + 1);
    unsafe {
        std::ptr::copy(
            list.arr.add(((end + 1) * vtable.type_size) as usize),
            list.arr.add((start * vtable.type_size) as usize),
            ((list.len - end - 1) * vtable.type_size) as usize,
        );
    }
    list.len = new_len;
}

#[unsafe(no_mangle)]
pub extern "C" fn efficient_list_insert(
    list: &mut DDPList<c_void>,
    index: DDPInt,
    elem: *mut c_void,
    any: &DDPAny,
) {
    if index < 0 || index > list.len {
        ddp_panic(
            1,
            format!(
                "Index außerhalb der Listen Länge (Index war {index}, Listen Länge war {})",
                list.len
            ),
        );
    }

    let vtable = any.get_generic_vtable();

    list.grow_if_needed(vtable.type_size, 1);
    unsafe {
        std::ptr::copy(
            list.arr.add((index * vtable.type_size) as usize),
            list.arr.add(((index + 1) * vtable.type_size) as usize),
            ((list.len - index) * vtable.type_size) as usize,
        );
        std::ptr::copy_nonoverlapping(
            elem,
            list.arr.add((index * vtable.type_size) as usize),
            vtable.type_size as usize,
        );
    }
    list.len += 1;

    claim_non_primitive(vtable, elem, any);
}

#[unsafe(no_mangle)]
pub extern "C" fn efficient_list_insert_range(
    list: &mut DDPList<c_void>,
    index: DDPInt,
    other: &mut DDPList<c_void>,
    any: &DDPAny,
) {
    if index < 0 || index > list.len {
        ddp_panic(
            1,
            format!(
                "Index außerhalb der Listen Länge (Index war {index}, Listen Länge war {})",
                list.len
            ),
        );
    }

    let vtable = any.get_generic_vtable();
    let new_len = list.len + other.len;

    list.grow_if_needed(vtable.type_size, other.len);
    unsafe {
        std::ptr::copy(
            list.arr.add((index * vtable.type_size) as usize),
            list.arr
                .add(((index + other.len) * vtable.type_size) as usize),
            ((list.len - index) * vtable.type_size) as usize,
        );
        std::ptr::copy_nonoverlapping(
            other.arr,
            list.arr.add((index * vtable.type_size) as usize),
            (other.len * vtable.type_size) as usize,
        );
    }
    list.len = new_len;

    if vtable.deep_copy_func.is_some() {
        for i in 0..other.len {
            unsafe {
                claim_non_primitive(vtable, other.arr.add((i * vtable.type_size) as usize), any);
            }
        }
    }
}

#[unsafe(no_mangle)]
pub extern "C" fn Aneinandergehaengt_Buchstabe_Ref(ret: *mut DDPString, liste: &DDPList<DDPChar>) {
    unsafe {
        let slice = std::slice::from_raw_parts(liste.arr, liste.len as usize);
        std::ptr::write(ret, DDPString::from(slice));
    }
}
